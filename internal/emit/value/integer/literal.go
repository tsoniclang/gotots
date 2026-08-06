package integer

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Literal(
	context api.Context,
	sourceType types.Type,
	decimal string,
) (tsgo.Expression, error) {
	carrier, ok := Describe(context.TypesSizes(), sourceType)
	if !ok {
		return nil, &api.IntegerCarrierError{
			Carrier: api.IntegerCarrierInvalid,
		}
	}
	return CarrierLiteral(context, carrier, decimal)
}

func CarrierLiteral(
	context api.Context,
	carrier Carrier,
	decimal string,
) (tsgo.Expression, error) {
	abi, err := api.NewScalarABIFromSizes(
		context.IntegerRepresentation(),
		context.TypesSizes(),
	)
	if err != nil {
		return nil, err
	}
	return api.IntegerLiteral(
		context.Factory(),
		abi,
		carrier.Alias(),
		decimal,
	)
}

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
	target, err := CarrierLiteral(context, carrier, magnitude)
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
