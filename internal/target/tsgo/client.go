package tsgo

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type ClientError struct {
	Operation string
	Reason    string
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("TS-Go client %s: %s", e.Operation, e.Reason)
}

type RemoteError struct {
	Operation string
	Code      int
	Message   string
}

func (e *RemoteError) Error() string {
	return fmt.Sprintf(
		"TS-Go %s error [%d]: %s",
		e.Operation,
		e.Code,
		e.Message,
	)
}

type Client struct {
	mutex     sync.Mutex
	tool      Tool
	command   *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	writer    *bufio.Writer
	reader    *bufio.Reader
	stderr    bytes.Buffer
	requestID uint64
	closed    bool
}

func StartClientWithTool(tool Tool, workingDirectory string) (*Client, error) {
	if !tool.Valid() {
		return nil, &ClientError{Operation: "start", Reason: "TS-Go tool is invalid"}
	}
	var err error
	workingDirectory, err = filepath.Abs(workingDirectory)
	if err != nil {
		return nil, &ClientError{Operation: "start", Reason: err.Error()}
	}
	info, err := os.Stat(workingDirectory)
	if err != nil {
		return nil, &ClientError{Operation: "start", Reason: err.Error()}
	}
	if !info.IsDir() {
		return nil, &ClientError{Operation: "start", Reason: workingDirectory + " is not a directory"}
	}

	command, err := tool.command(
		context.Background(),
		"--api", "--async", "--cwd", workingDirectory,
	)
	if err != nil {
		return nil, err
	}
	client := &Client{tool: tool, command: command}
	command.Stderr = &client.stderr
	client.stdin, err = command.StdinPipe()
	if err != nil {
		return nil, &ClientError{Operation: "open stdin", Reason: err.Error()}
	}
	client.stdout, err = command.StdoutPipe()
	if err != nil {
		client.stdin.Close()
		return nil, &ClientError{Operation: "open stdout", Reason: err.Error()}
	}
	client.writer = bufio.NewWriter(client.stdin)
	client.reader = bufio.NewReader(client.stdout)
	if err := command.Start(); err != nil {
		client.stdin.Close()
		client.stdout.Close()
		return nil, &ClientError{Operation: "start", Reason: err.Error()}
	}
	if err := tool.Verify(); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		client.stdin.Close()
		client.stdout.Close()
		return nil, err
	}
	return client, nil
}

func (c *Client) print(params printNodeParams) (string, error) {
	encoded, err := json.Marshal(params)
	if err != nil {
		return "", &ClientError{
			Operation: "encode printNode request",
			Reason:    err.Error(),
		}
	}
	result, err := c.request("printNode", encoded)
	if err != nil {
		return "", err
	}
	var printed string
	if err := json.Unmarshal(result, &printed); err != nil {
		return "", &ClientError{
			Operation: "decode printNode result",
			Reason:    err.Error(),
		}
	}
	return printed, nil
}

func (c *Client) request(
	operation string,
	params json.RawMessage,
) (json.RawMessage, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.closed {
		return nil, &ClientError{
			Operation: operation,
			Reason:    "client is closed",
		}
	}
	c.requestID++
	request := rpcRequest{
		JSONRPC: "2.0",
		ID:      c.requestID,
		Method:  operation,
		Params:  params,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, &ClientError{
			Operation: "encode " + operation + " request",
			Reason:    err.Error(),
		}
	}
	if _, err := fmt.Fprintf(c.writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return nil, &ClientError{
			Operation: "write " + operation + " header",
			Reason:    err.Error(),
		}
	}
	if _, err := c.writer.Write(payload); err != nil {
		return nil, &ClientError{
			Operation: "write " + operation + " payload",
			Reason:    err.Error(),
		}
	}
	if err := c.writer.Flush(); err != nil {
		return nil, &ClientError{
			Operation: "flush " + operation + " request",
			Reason:    err.Error(),
		}
	}

	contentLength, err := readContentLength(c.reader)
	if err != nil {
		return nil, &ClientError{
			Operation: "read " + operation + " response header",
			Reason:    err.Error(),
		}
	}
	limited := &io.LimitedReader{R: c.reader, N: int64(contentLength)}
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	var response rpcResponse
	if err := decoder.Decode(&response); err != nil {
		return nil, &ClientError{
			Operation: "decode " + operation + " response",
			Reason:    err.Error(),
		}
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, &ClientError{
			Operation: "decode " + operation + " response",
			Reason:    err.Error(),
		}
	}
	if limited.N != 0 {
		return nil, &ClientError{
			Operation: "decode " + operation + " response",
			Reason:    fmt.Sprintf("%d unread bytes", limited.N),
		}
	}
	if response.JSONRPC != "2.0" || response.ID != c.requestID {
		return nil, &ClientError{
			Operation: "correlate " + operation + " response",
			Reason: fmt.Sprintf(
				"jsonrpc=%q id=%d, want jsonrpc=%q id=%d",
				response.JSONRPC,
				response.ID,
				"2.0",
				c.requestID,
			),
		}
	}
	if response.Error != nil {
		return nil, &RemoteError{
			Operation: operation,
			Code:      response.Error.Code,
			Message:   response.Error.Message,
		}
	}
	if response.Result == nil {
		return nil, &ClientError{
			Operation: "decode " + operation + " response",
			Reason:    "result is absent",
		}
	}
	return slices.Clone(response.Result), nil
}

func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	closeErr := c.stdin.Close()
	waitErr := c.command.Wait()
	c.stdout.Close()
	verifyErr := c.tool.Verify()
	var result []error
	if closeErr != nil {
		result = append(result, &ClientError{Operation: "close stdin", Reason: closeErr.Error()})
	}
	if waitErr != nil {
		reason := waitErr.Error()
		if stderr := strings.TrimSpace(c.stderr.String()); stderr != "" {
			reason += ": " + stderr
		}
		result = append(result, &ClientError{Operation: "wait", Reason: reason})
	}
	result = append(result, verifyErr)
	return errors.Join(result...)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func readContentLength(reader *bufio.Reader) (int, error) {
	contentLength := -1
	for headerCount := 0; ; headerCount++ {
		if headerCount == 32 {
			return 0, fmt.Errorf("too many response headers")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			break
		}
		name, value, found := strings.Cut(line, ":")
		if !found {
			return 0, fmt.Errorf("malformed header %q", line)
		}
		if !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			continue
		}
		if contentLength >= 0 {
			return 0, fmt.Errorf("duplicate Content-Length")
		}
		parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 63)
		if err != nil || uint64(int(parsed)) != parsed {
			return 0, fmt.Errorf("invalid Content-Length %q", value)
		}
		contentLength = int(parsed)
	}
	if contentLength < 0 {
		return 0, fmt.Errorf("Content-Length is absent")
	}
	return contentLength, nil
}
