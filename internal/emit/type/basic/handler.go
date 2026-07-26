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
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	var alias api.PrimitiveAlias
	switch basic.Kind() {
	case types.Bool:
		alias = api.PrimitiveBool
	case types.Int64:
		alias = api.PrimitiveInt64
	case types.Int32:
		alias = api.PrimitiveInt32
	case types.Int:
		switch context.TypesSizes().Sizeof(types.Typ[types.Int]) {
		case 4:
			alias = api.PrimitiveInt32
		case 8:
			alias = api.PrimitiveInt64
		default:
			return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
		}
	default:
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

func SupportsExactInt32(sizes types.Sizes, sourceType types.Type) bool {
	if sizes == nil || sourceType == nil {
		return false
	}
	basic, ok := types.Unalias(sourceType).Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch basic.Kind() {
	case types.Int32:
		return true
	case types.Int:
		return sizes.Sizeof(types.Typ[types.Int]) == 4
	default:
		return false
	}
}
