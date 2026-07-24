package semantic

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

const semanticShardBufferBytes = 256 * 1024

type binaryShardEncoder struct {
	output  *bufio.Writer
	written int64
	err     error
	scratch [binary.MaxVarintLen64]byte
}

func newBinaryShardEncoder(output io.Writer) *binaryShardEncoder {
	return &binaryShardEncoder{
		output: bufio.NewWriterSize(output, semanticShardBufferBytes),
	}
}

func (encoder *binaryShardEncoder) raw(value []byte) {
	if encoder.err != nil {
		return
	}
	written, err := encoder.output.Write(value)
	encoder.written += int64(written)
	if err != nil {
		encoder.err = err
		return
	}
	if written != len(value) {
		encoder.err = io.ErrShortWrite
	}
}

func (encoder *binaryShardEncoder) unsigned(value uint64) {
	size := binary.PutUvarint(encoder.scratch[:], value)
	encoder.raw(encoder.scratch[:size])
}

func (encoder *binaryShardEncoder) signed(value int64) {
	size := binary.PutVarint(encoder.scratch[:], value)
	encoder.raw(encoder.scratch[:size])
}

func (encoder *binaryShardEncoder) boolean(value bool) {
	if value {
		encoder.unsigned(1)
		return
	}
	encoder.unsigned(0)
}

func (encoder *binaryShardEncoder) text(value string) {
	encoder.unsigned(uint64(len(value)))
	encoder.raw([]byte(value))
}

func (encoder *binaryShardEncoder) count(value int) {
	if value < 0 {
		if encoder.err == nil {
			encoder.err = fmt.Errorf(
				"semantic binary shard has negative count",
			)
		}
		return
	}
	encoder.unsigned(uint64(value))
}

func (encoder *binaryShardEncoder) finish() (int64, error) {
	if encoder.err != nil {
		return 0, encoder.err
	}
	if err := encoder.output.Flush(); err != nil {
		return 0, err
	}
	return encoder.written, nil
}

type binaryShardDecoder struct {
	input        *bufio.Reader
	expected     int64
	consumed     int64
	identityUses *binaryIdentityUses
}

func newBinaryShardDecoder(
	input io.Reader,
	expected int64,
) (*binaryShardDecoder, error) {
	if input == nil || expected <= 0 {
		return nil, fmt.Errorf(
			"semantic binary shard requires input and positive extent",
		)
	}
	return &binaryShardDecoder{
		input:    bufio.NewReaderSize(input, semanticShardBufferBytes),
		expected: expected,
	}, nil
}

func (decoder *binaryShardDecoder) ReadByte() (byte, error) {
	if decoder.consumed >= decoder.expected {
		return 0, io.EOF
	}
	value, err := decoder.input.ReadByte()
	if err != nil {
		return 0, err
	}
	decoder.consumed++
	return value, nil
}

func (decoder *binaryShardDecoder) raw(size int) ([]byte, error) {
	if size < 0 || int64(size) > decoder.remaining() {
		return nil, fmt.Errorf(
			"semantic binary shard field length %d exceeds remaining %d",
			size,
			decoder.remaining(),
		)
	}
	out := make([]byte, size)
	if _, err := io.ReadFull(decoder.input, out); err != nil {
		return nil, err
	}
	decoder.consumed += int64(size)
	return out, nil
}

func (decoder *binaryShardDecoder) remaining() int64 {
	return decoder.expected - decoder.consumed
}

func (decoder *binaryShardDecoder) unsigned(
	name string,
) (uint64, error) {
	value, err := binary.ReadUvarint(decoder)
	if err != nil {
		return 0, fmt.Errorf(
			"decode semantic binary %s: %w", name, err,
		)
	}
	return value, nil
}

func (decoder *binaryShardDecoder) signed(
	name string,
) (int64, error) {
	value, err := binary.ReadVarint(decoder)
	if err != nil {
		return 0, fmt.Errorf(
			"decode semantic binary %s: %w", name, err,
		)
	}
	return value, nil
}

func (decoder *binaryShardDecoder) boolean(
	name string,
) (bool, error) {
	value, err := decoder.unsigned(name)
	if err != nil {
		return false, err
	}
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf(
			"semantic binary %s boolean is %d", name, value,
		)
	}
}

func (decoder *binaryShardDecoder) text(name string) (string, error) {
	size, err := decoder.count(name + " bytes")
	if err != nil {
		return "", err
	}
	value, err := decoder.raw(size)
	if err != nil {
		return "", fmt.Errorf(
			"decode semantic binary %s: %w", name, err,
		)
	}
	return string(value), nil
}

func (decoder *binaryShardDecoder) count(name string) (int, error) {
	value, err := decoder.unsigned(name)
	if err != nil {
		return 0, err
	}
	maxInt := uint64(^uint(0) >> 1)
	if value > maxInt || value > uint64(decoder.remaining()) {
		return 0, fmt.Errorf(
			"semantic binary %s count %d exceeds remaining %d",
			name,
			value,
			decoder.remaining(),
		)
	}
	return int(value), nil
}

func (decoder *binaryShardDecoder) finish() error {
	if decoder.consumed != decoder.expected {
		return fmt.Errorf(
			"semantic binary shard has %d trailing bytes",
			decoder.expected-decoder.consumed,
		)
	}
	return nil
}

func readUnsignedAs[Value ~uint8 | ~uint16 | ~uint64](
	decoder *binaryShardDecoder,
	name string,
) (Value, error) {
	value, err := decoder.unsigned(name)
	if err != nil {
		return 0, err
	}
	maximum := uint64(^Value(0))
	if value > maximum {
		return 0, fmt.Errorf(
			"semantic binary %s value %d exceeds domain", name, value,
		)
	}
	return Value(value), nil
}

func readSignedInt(
	decoder *binaryShardDecoder,
	name string,
) (int, error) {
	value, err := decoder.signed(name)
	if err != nil {
		return 0, err
	}
	converted := int(value)
	if int64(converted) != value {
		return 0, fmt.Errorf(
			"semantic binary %s value %d exceeds int", name, value,
		)
	}
	return converted, nil
}

func writeBinaryRecords[Record any](
	encoder *binaryShardEncoder,
	records []Record,
	write func(*binaryShardEncoder, Record),
) {
	encoder.count(len(records))
	for _, record := range records {
		write(encoder, record)
	}
}

func readBinaryRecords[Record any](
	decoder *binaryShardDecoder,
	name string,
	read func(*binaryShardDecoder) (Record, error),
) ([]Record, error) {
	count, err := decoder.count(name)
	if err != nil {
		return nil, err
	}
	out := make([]Record, count)
	for index := range out {
		out[index], err = read(decoder)
		if err != nil {
			return nil, fmt.Errorf(
				"decode semantic binary %s record %d: %w",
				name,
				index,
				err,
			)
		}
	}
	return out, nil
}
