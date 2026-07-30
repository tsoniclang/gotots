package namedstruct

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func operationFieldValue(
	context api.Context,
	source ast.Node,
	receiver string,
	field field,
	canonicalStorage bool,
) (api.ExpressionEmission, error) {
	value := property(context, receiver, field.name)
	if !canonicalStorage {
		return api.DirectExpression(value), nil
	}
	value = context.Factory().PropertyAccessExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(receiver),
			nil,
			context.Factory().Identifier("$storage"),
			tsgo.NodeFlagsNone,
		),
		nil,
		context.Factory().Identifier(field.name),
		tsgo.NodeFlagsNone,
	)
	return context.Values().FromStorage(
		context,
		source,
		field.object.Type(),
		api.DirectExpression(value),
	)
}

func operationConstructionValue(
	context api.Context,
	source ast.Node,
	field field,
	value api.ExpressionEmission,
	canonicalStorage bool,
) (api.ExpressionEmission, error) {
	if !canonicalStorage {
		return value, nil
	}
	return context.Values().ToStorage(
		context,
		source,
		field.object.Type(),
		value,
	)
}
