package integer

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func NormalizeFixedWidth(
	context api.Context,
	carrier Carrier,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	representation, ok := CarrierRepresentation(
		context.IntegerRepresentation(),
		carrier,
	)
	if !ok {
		return api.ExpressionEmission{}, &api.IntegerRepresentationError{
			Representation: context.IntegerRepresentation(),
		}
	}
	switch representation {
	case api.IntegerCarrierBigInt:
		return normalizeBigInt(context, carrier, value)
	case api.IntegerCarrierNumber:
		return api.DirectExpression(
			normalizeNumber(context.Factory(), carrier, value),
		), nil
	default:
		return api.ExpressionEmission{}, &api.IntegerCarrierError{
			Carrier: representation,
		}
	}
}

func normalizeBigInt(
	context api.Context,
	carrier Carrier,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if carrier.Width() != 64 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "bigint carrier does not have 64-bit width",
		}
	}
	symbol := api.RuntimeIntegerNormalizeUnsigned64
	if carrier.Signed() {
		symbol = api.RuntimeIntegerNormalizeSigned64
	}
	reference, err := context.Names().Runtime(symbol, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		),
		reference.Requests()...,
	), nil
}

func normalizeNumber(
	factory tsgo.Factory,
	carrier Carrier,
	value tsgo.Expression,
) tsgo.Expression {
	if carrier.Width() > 32 {
		return value
	}
	zero := factory.NumericLiteral("0", tsgo.TokenFlagsNone)
	if carrier.Width() == 32 {
		operator := tsgo.BinaryOperatorBarToken
		if !carrier.Signed() {
			operator = tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken
		}
		return factory.BinaryExpression(
			nil,
			value,
			nil,
			factory.BinaryOperatorToken(operator),
			zero,
		)
	}
	if carrier.Signed() {
		distance := factory.NumericLiteral(
			fmt.Sprintf("%d", 32-carrier.Width()),
			tsgo.TokenFlagsNone,
		)
		return factory.BinaryExpression(
			nil,
			factory.BinaryExpression(
				nil,
				value,
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorLessThanLessThanToken,
				),
				distance,
			),
			nil,
			factory.BinaryOperatorToken(
				tsgo.BinaryOperatorGreaterThanGreaterThanToken,
			),
			distance,
		)
	}
	mask := uint64(1)<<carrier.Width() - 1
	return factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorAmpersandToken),
		factory.NumericLiteral(
			fmt.Sprintf("%d", mask),
			tsgo.TokenFlagsNone,
		),
	)
}
