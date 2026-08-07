package tsgo

import (
	"bytes"
	"encoding/base64"
)

type PrintOptions struct {
	PreserveSourceNewlines        bool
	NeverASCIIEscape              bool
	TerminateUnterminatedLiterals bool
}

type printNodeParams struct {
	Data                          string `json:"data"`
	PreserveSourceNewlines        bool   `json:"preserveSourceNewlines,omitempty"`
	NeverASCIIEscape              bool   `json:"neverAsciiEscape,omitempty"`
	TerminateUnterminatedLiterals bool   `json:"terminateUnterminatedLiterals,omitempty"`
}

func (c *Client) PrintNode(node Node, options PrintOptions) (string, error) {
	data, err := EncodeNode(node)
	if err != nil {
		return "", err
	}
	return c.print(printNodeParams{
		Data:                          base64.StdEncoding.EncodeToString(data),
		PreserveSourceNewlines:        options.PreserveSourceNewlines,
		NeverASCIIEscape:              options.NeverASCIIEscape,
		TerminateUnterminatedLiterals: options.TerminateUnterminatedLiterals,
	})
}

// PrintEncodedSourceFile prints one source file represented by the pinned
// TS-Go external-AST protocol. The caller owns AST construction; this method
// performs no parsing or semantic recovery.
func (c *Client) PrintEncodedSourceFile(data []byte, options PrintOptions) (string, error) {
	if c == nil {
		return "", &ClientError{Operation: "print encoded source file", Reason: "client is nil"}
	}
	if len(data) < headerSize {
		return "", &ClientError{Operation: "print encoded source file", Reason: "protocol payload is truncated"}
	}
	if !bytes.Equal(data[:4], []byte{0, 0, 0, protocolVersion}) {
		return "", &ClientError{Operation: "print encoded source file", Reason: "protocol version is not pinned"}
	}
	return c.print(printNodeParams{
		Data:                          base64.StdEncoding.EncodeToString(data),
		PreserveSourceNewlines:        options.PreserveSourceNewlines,
		NeverASCIIEscape:              options.NeverASCIIEscape,
		TerminateUnterminatedLiterals: options.TerminateUnterminatedLiterals,
	})
}
