package api

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type IntegerRepresentation uint8

const (
	IntegerRepresentationInvalid IntegerRepresentation = iota
	IntegerRepresentationNumber
	IntegerRepresentationBigInt
)

func (r IntegerRepresentation) Valid() bool {
	return r == IntegerRepresentationNumber ||
		r == IntegerRepresentationBigInt
}

func (r IntegerRepresentation) String() string {
	switch r {
	case IntegerRepresentationNumber:
		return "number"
	case IntegerRepresentationBigInt:
		return "bigint"
	default:
		return fmt.Sprintf("integer-representation(%d)", r)
	}
}

type EvaluationOrder uint8

const (
	EvaluationOrderInvalid EvaluationOrder = iota
	EvaluationOrderDirect
	EvaluationOrderPreserveGo
)

func (o EvaluationOrder) Valid() bool {
	return o == EvaluationOrderDirect ||
		o == EvaluationOrderPreserveGo
}

func (o EvaluationOrder) String() string {
	switch o {
	case EvaluationOrderDirect:
		return "direct"
	case EvaluationOrderPreserveGo:
		return "preserve-go"
	default:
		return fmt.Sprintf("evaluation-order(%d)", o)
	}
}

func IntegerLiteral(
	factory tsgo.Factory,
	representation IntegerRepresentation,
	decimal string,
) (tsgo.Expression, error) {
	if decimal == "" {
		return nil, &IntegerRepresentationError{
			Representation: representation,
		}
	}
	switch representation {
	case IntegerRepresentationNumber:
		return factory.NumericLiteral(decimal, tsgo.TokenFlagsNone), nil
	case IntegerRepresentationBigInt:
		return factory.BigIntLiteral(decimal+"n", tsgo.TokenFlagsNone), nil
	default:
		return nil, &IntegerRepresentationError{
			Representation: representation,
		}
	}
}

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
	PrimitiveString
	PrimitiveFloat32
	PrimitiveFloat64
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
	case PrimitiveString:
		return name, tsgo.KeywordTypeSyntaxKindStringKeyword, nil
	case PrimitiveFloat32, PrimitiveFloat64:
		return name, tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
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
	case PrimitiveString:
		return "gostring", nil
	case PrimitiveFloat32:
		return "float32", nil
	case PrimitiveFloat64:
		return "float64", nil
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
