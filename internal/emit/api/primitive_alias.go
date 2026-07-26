package api

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
