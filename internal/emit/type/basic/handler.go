package basic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func Emit(context api.Context, source ast.Expr) (api.TypeEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	return EmitRepresented(context, source, sourceType)
}

func EmitRepresented(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	alias, ok := PrimitiveAlias(context.TypesSizes(), sourceType)
	if !ok {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().Primitive(alias)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			nil,
		),
		reference.Requests()...,
	), nil
}

func SupportsInteger(sizes types.Sizes, sourceType types.Type) bool {
	_, ok := integervalue.Describe(sizes, sourceType)
	return ok
}

func PrimitiveAlias(
	sizes types.Sizes,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	if sourceType != nil {
		if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
			basic.Kind() == types.Bool {
			return api.PrimitiveBool, true
		}
	}
	carrier, ok := integervalue.Describe(sizes, sourceType)
	if !ok {
		return api.PrimitiveInvalid, false
	}
	return carrier.Alias(), true
}
