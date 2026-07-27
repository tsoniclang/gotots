package api

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type PrimitiveAlias uint8

const (
	PrimitiveInvalid PrimitiveAlias = iota
	PrimitiveBool
	PrimitiveInt8
	PrimitiveInt16
	PrimitiveInt32
	PrimitiveInt64
	PrimitiveUint8
	PrimitiveUint16
	PrimitiveUint32
	PrimitiveUint64
)

func PrimitiveAliasRepresentation(
	alias PrimitiveAlias,
	integer IntegerRepresentation,
) (string, tsgo.KeywordTypeSyntaxKind, error) {
	name, err := PrimitiveAliasName(alias)
	if err != nil {
		return "", 0, err
	}
	switch alias {
	case PrimitiveBool:
		return name, tsgo.KeywordTypeSyntaxKindBooleanKeyword, nil
	case PrimitiveInt8,
		PrimitiveInt16,
		PrimitiveInt32,
		PrimitiveInt64,
		PrimitiveUint8,
		PrimitiveUint16,
		PrimitiveUint32,
		PrimitiveUint64:
		keyword, err := integerKeyword(integer)
		return name, keyword, err
	default:
		return "", 0, &PrimitiveAliasError{Alias: alias}
	}
}

func PrimitiveAliasName(alias PrimitiveAlias) (string, error) {
	switch alias {
	case PrimitiveBool:
		return "bool", nil
	case PrimitiveInt8:
		return "int8", nil
	case PrimitiveInt16:
		return "int16", nil
	case PrimitiveInt32:
		return "int32", nil
	case PrimitiveInt64:
		return "int64", nil
	case PrimitiveUint8:
		return "uint8", nil
	case PrimitiveUint16:
		return "uint16", nil
	case PrimitiveUint32:
		return "uint32", nil
	case PrimitiveUint64:
		return "uint64", nil
	default:
		return "", &PrimitiveAliasError{Alias: alias}
	}
}

func integerKeyword(
	representation IntegerRepresentation,
) (tsgo.KeywordTypeSyntaxKind, error) {
	switch representation {
	case IntegerRepresentationNumber:
		return tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	case IntegerRepresentationBigInt:
		return tsgo.KeywordTypeSyntaxKindBigIntKeyword, nil
	default:
		return 0, &IntegerRepresentationError{Representation: representation}
	}
}

type PrimitiveAliasError struct {
	Alias PrimitiveAlias
}

func (e *PrimitiveAliasError) Error() string {
	return fmt.Sprintf("primitive alias %d is invalid", e.Alias)
}

type IntegerRepresentationError struct {
	Representation IntegerRepresentation
}

func (e *IntegerRepresentationError) Error() string {
	return fmt.Sprintf(
		"integer representation %d is invalid",
		e.Representation,
	)
}
