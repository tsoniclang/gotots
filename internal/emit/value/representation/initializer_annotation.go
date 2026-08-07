package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

func (Owner) RequiresInitializerTypeAnnotation(
	context api.Context,
	source ast.Expr,
	targetType types.Type,
) (bool, error) {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			break
		}
		source = parenthesized.X
	}
	call, ok := source.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return false, nil
	}
	callee, ok := context.TypesInfo().TypeAndValue(call.Fun)
	if !ok || !callee.IsType() ||
		!types.Identical(context.TypesInfo().TypeOf(call), targetType) {
		return false, nil
	}
	if _, ok := types.Unalias(targetType).(*types.Basic); !ok {
		return false, nil
	}
	sourceModel, ok := definedtype.ResolveBasic(
		context.TypesInfo().TypeOf(call.Args[0]),
	)
	if !ok {
		return false, nil
	}
	representation, err := sourceModel.Representation(context)
	if err != nil {
		return false, err
	}
	return representation.Kind() ==
		api.DefinedValueRepresentationGeneratedNumeric, nil
}
