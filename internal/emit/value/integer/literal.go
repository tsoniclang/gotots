package integer

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitConstant(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value constant.Value,
) (api.ExpressionEmission, error) {
	carrier, ok := Describe(context.TypesSizes(), targetType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	magnitude, negative, ok := FormatConstant(
		context.IntegerRepresentation(),
		carrier,
		value,
	)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := api.IntegerLiteral(
		context.Factory(),
		context.IntegerRepresentation(),
		magnitude,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if negative {
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			target,
		)
	}
	return api.DirectExpression(target), nil
}
