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
) (string, tsgo.KeywordTypeSyntaxKind, error) {
	switch alias {
	case PrimitiveBool:
		return "bool", tsgo.KeywordTypeSyntaxKindBooleanKeyword, nil
	case PrimitiveInt32:
		return "int32", tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	case PrimitiveInt64:
		return "int64", tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	default:
		return "", 0, &PrimitiveAliasError{Alias: alias}
	}
}

type PrimitiveAliasError struct {
	Alias PrimitiveAlias
}

func (e *PrimitiveAliasError) Error() string {
	return fmt.Sprintf("primitive alias %d is invalid", e.Alias)
}
