package identifier

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.Ident,
) (api.ExpressionEmission, error) {
	object := context.TypesInfo().Uses[source]
	if object == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch object {
	case types.Universe.Lookup("false"):
		return emitBooleanConstant(context, source, context.Factory().FalseLiteral())
	case types.Universe.Lookup("true"):
		return emitBooleanConstant(context, source, context.Factory().TrueLiteral())
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

func emitBooleanConstant(
	context api.Context,
	source *ast.Ident,
	literal tsgo.Expression,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	targetType := context.ExpectedType()
	if sourceType == nil ||
		targetType == nil ||
		!types.AssignableTo(sourceType, targetType) ||
		!isBoolean(targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.DirectExpression(literal), nil
}

func isBoolean(source types.Type) bool {
	basic, ok := types.Unalias(source).(*types.Basic)
	return ok && basic.Kind() == types.Bool
}
