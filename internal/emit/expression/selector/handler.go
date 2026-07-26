package selector

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	source *ast.SelectorExpr,
) (api.ExpressionEmission, error) {
	if context.TypesInfo().Selections[source] != nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	qualifier, ok := source.X.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	packageName, ok := context.TypesInfo().Uses[qualifier].(*types.PkgName)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	object, ok := context.TypesInfo().Uses[source.Sel].(*types.Const)
	if !ok || object.Pkg() != packageName.Imported() {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().Identifier(reference.Name()),
		reference.Requests()...,
	), nil
}
