package tsgoprinter

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	inputMagic       = "TSTSPR01"
	outputMagic      = "TSTSPR02"
	maximumFileCount = 1_000_000
	maximumFrameSize = 1 << 30
)

type Config struct {
	ModuleDirectory  string
	WorkingDirectory string
	PrintOptions     tsgo.PrintOptions
}

func Run(config Config, input io.Reader, output io.Writer) error {
	if input == nil || output == nil {
		return fmt.Errorf("run TS-Go AST printer: input or output is nil")
	}
	reader := bufio.NewReader(input)
	magic := make([]byte, len(inputMagic))
	if _, err := io.ReadFull(reader, magic); err != nil {
		return fmt.Errorf("read TS-Go AST printer header: %w", err)
	}
	if string(magic) != inputMagic {
		return fmt.Errorf("read TS-Go AST printer header: invalid magic")
	}
	count, err := readUint32(reader)
	if err != nil {
		return fmt.Errorf("read TS-Go AST printer file count: %w", err)
	}
	if count > maximumFileCount {
		return fmt.Errorf("read TS-Go AST printer file count: %d exceeds limit", count)
	}
	client, err := tsgo.StartClient(config.ModuleDirectory, config.WorkingDirectory)
	if err != nil {
		return err
	}
	defer client.Close()

	writer := bufio.NewWriter(output)
	if _, err := io.WriteString(writer, outputMagic); err != nil {
		return fmt.Errorf("write TS-Go AST printer header: %w", err)
	}
	if err := writeUint32(writer, count); err != nil {
		return fmt.Errorf("write TS-Go AST printer file count: %w", err)
	}
	for index := uint32(0); index < count; index++ {
		payload, err := readFrame(reader)
		if err != nil {
			return fmt.Errorf("read TS-Go AST frame %d: %w", index, err)
		}
		printed, err := client.PrintEncodedSourceFile(payload, config.PrintOptions)
		if err != nil {
			return fmt.Errorf("print TS-Go AST frame %d: %w", index, err)
		}
		if err := writeFrame(writer, []byte(printed)); err != nil {
			return fmt.Errorf("write TS-Go source frame %d: %w", index, err)
		}
	}
	if extra, err := reader.ReadByte(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("read TS-Go AST printer input: trailing byte 0x%02x", extra)
		}
		return fmt.Errorf("read TS-Go AST printer input: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush TS-Go AST printer output: %w", err)
	}
	return nil
}

func readFrame(reader io.Reader) ([]byte, error) {
	length, err := readUint32(reader)
	if err != nil {
		return nil, err
	}
	if length > maximumFrameSize {
		return nil, fmt.Errorf("frame size %d exceeds limit", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeFrame(writer io.Writer, payload []byte) error {
	if len(payload) > maximumFrameSize {
		return fmt.Errorf("frame size %d exceeds limit", len(payload))
	}
	if err := writeUint32(writer, uint32(len(payload))); err != nil {
		return err
	}
	_, err := writer.Write(payload)
	return err
}

func readUint32(reader io.Reader) (uint32, error) {
	var encoded [4]byte
	if _, err := io.ReadFull(reader, encoded[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(encoded[:]), nil
}

func writeUint32(writer io.Writer, value uint32) error {
	var encoded [4]byte
	binary.LittleEndian.PutUint32(encoded[:], value)
	_, err := writer.Write(encoded[:])
	return err
}
