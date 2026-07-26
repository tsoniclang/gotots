package api

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func PrimitiveAliasFor(
	sizes types.Sizes,
	sourceType types.Type,
) (PrimitiveAlias, bool) {
	if sizes == nil || sourceType == nil {
		return PrimitiveInvalid, false
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return PrimitiveInvalid, false
	}
	switch basic.Kind() {
	case types.Bool:
		return PrimitiveBool, true
	case types.Int32:
		return PrimitiveInt32, true
	case types.Int64:
		return PrimitiveInt64, true
	case types.Int:
		switch sizes.Sizeof(types.Typ[types.Int]) {
		case 4:
			return PrimitiveInt32, true
		case 8:
			return PrimitiveInt64, true
		}
	}
	return PrimitiveInvalid, false
}

type PrimitiveAlias uint8

const (
	PrimitiveInvalid PrimitiveAlias = iota
	PrimitiveBool
	PrimitiveInt32
	PrimitiveInt64
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
	case PrimitiveInt32, PrimitiveInt64:
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
	case PrimitiveInt32:
		return "int32", nil
	case PrimitiveInt64:
		return "int64", nil
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
