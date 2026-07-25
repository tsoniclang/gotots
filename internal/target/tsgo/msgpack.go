package tsgo

import (
	"fmt"
	"math"
)

const noStructuredData uint32 = math.MaxUint32

type structuredResult struct {
	data   []byte
	offset uint32
}

func appendFileReferences(data []byte, references []FileReference) (structuredResult, error) {
	if len(references) == 0 {
		return structuredResult{data: data, offset: noStructuredData}, nil
	}
	if len(data) > math.MaxUint32 {
		return structuredResult{}, &EncodeError{Reason: "structured data exceeds uint32 size"}
	}
	result := structuredResult{data: data, offset: uint32(len(data))}
	result.data = appendArrayHeader(result.data, len(references))
	for _, reference := range references {
		result.data = appendArrayHeader(result.data, 5)
		result.data = appendMessagePackUint(result.data, reference.Pos)
		result.data = appendMessagePackUint(result.data, reference.End)
		result.data = appendMessagePackString(result.data, reference.FileName)
		result.data = appendMessagePackUint(result.data, reference.ResolutionMode)
		result.data = appendMessagePackBool(result.data, reference.Preserve)
	}
	return result, nil
}

func appendArrayHeader(data []byte, length int) []byte {
	switch {
	case length <= 0x0f:
		return append(data, byte(0x90|length))
	case length <= math.MaxUint16:
		return append(data, 0xdc, byte(length>>8), byte(length))
	default:
		return append(
			data,
			0xdd,
			byte(length>>24),
			byte(length>>16),
			byte(length>>8),
			byte(length),
		)
	}
}

func appendMessagePackUint(data []byte, value uint32) []byte {
	switch {
	case value <= 0x7f:
		return append(data, byte(value))
	case value <= math.MaxUint8:
		return append(data, 0xcc, byte(value))
	case value <= math.MaxUint16:
		return append(data, 0xcd, byte(value>>8), byte(value))
	default:
		return append(data, 0xce, byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
	}
}

func appendMessagePackString(data []byte, value string) []byte {
	length := len(value)
	switch {
	case length <= 0x1f:
		data = append(data, byte(0xa0|length))
	case length <= math.MaxUint8:
		data = append(data, 0xd9, byte(length))
	case length <= math.MaxUint16:
		data = append(data, 0xda, byte(length>>8), byte(length))
	case uint64(length) <= math.MaxUint32:
		data = append(
			data,
			0xdb,
			byte(length>>24),
			byte(length>>16),
			byte(length>>8),
			byte(length),
		)
	default:
		panic(fmt.Sprintf("message-pack string exceeds uint32 size: %d", length))
	}
	return append(data, value...)
}

func appendMessagePackBool(data []byte, value bool) []byte {
	if value {
		return append(data, 0xc3)
	}
	return append(data, 0xc2)
}
