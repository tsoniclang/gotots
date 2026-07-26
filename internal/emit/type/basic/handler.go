package basic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	alias, ok := api.PrimitiveAliasFor(context.TypesSizes(), sourceType)
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
	alias, ok := api.PrimitiveAliasFor(sizes, sourceType)
	return ok && (alias == api.PrimitiveInt32 || alias == api.PrimitiveInt64)
}

func SupportsString(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok && basic.Kind() == types.String
}

func SupportsStringIndex(sizes types.Sizes, sourceType types.Type) bool {
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 &&
		basic.Info()&types.IsInteger != 0 {
		sourceType = types.Typ[types.Int]
	}
	alias, ok := api.PrimitiveAliasFor(sizes, sourceType)
	return ok &&
		(alias == api.PrimitiveUint8 ||
			alias == api.PrimitiveInt32 ||
			alias == api.PrimitiveInt64)
}
