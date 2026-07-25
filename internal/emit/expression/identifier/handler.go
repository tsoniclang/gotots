package identifier

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	source *ast.Ident,
) (api.ExpressionEmission, error) {
	object := context.TypesInfo().Uses[source]
	if object == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch object {
	case types.Universe.Lookup("false"):
		return api.DirectExpression(context.Factory().FalseLiteral()), nil
	case types.Universe.Lookup("true"):
		return api.DirectExpression(context.Factory().TrueLiteral()), nil
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
