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
	var targetName string
	switch basic.Kind() {
	case types.Bool:
		targetName = "bool"
	case types.Int64:
		targetName = "int64"
	case types.Int:
		switch context.TypesSizes().Sizeof(types.Typ[types.Int]) {
		case 4:
			targetName = "int32"
		case 8:
			targetName = "int64"
		default:
			return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
		}
	default:
		return api.TypeEmission{}, api.Unsupported(context, api.CategoryType, source)
	}
	reference, err := context.Names().TypeImport("@tsonic/core/types.js", targetName)
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

func SupportsSignedArithmetic(sizes types.Sizes, sourceType types.Type) bool {
	if sizes == nil || sourceType == nil {
		return false
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return false
	}
	switch basic.Kind() {
	case types.Int64:
		return true
	case types.Int:
		return sizes.Sizeof(types.Typ[types.Int]) == 8
	default:
		return false
	}
}
