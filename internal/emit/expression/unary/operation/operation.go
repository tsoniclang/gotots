package operation

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	operator token.Token,
	sourceType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if carrier, ok := complexvalue.Describe(sourceType); ok {
		return complexOperation(context, operator, carrier, operand)
	}
	if carrier, ok := integervalue.Describe(
		context.TypesSizes(),
		sourceType,
	); ok {
		return integerOperation(context, operator, carrier, operand)
	}
	if _, ok := floatvalue.Describe(sourceType); ok {
		return floatOperation(context, operator, operand)
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok ||
		basic.Info()&types.IsBoolean == 0 ||
		operator != token.NOT {
		return api.ExpressionEmission{}, false, nil
	}
	result, err := api.NewExpressionEmission(
		operand.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			operand.Value(),
		),
		operand.Requests(),
	)
	return result, true, err
}

func complexOperation(
	context api.Context,
	operator token.Token,
	carrier complexvalue.Carrier,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if operator == token.ADD {
		result, err := api.NewExpressionEmission(
			operand.Before(),
			operand.Value(),
			operand.Requests(),
		)
		return result, true, err
	}
	if operator != token.SUB {
		return api.ExpressionEmission{}, false, nil
	}
	symbol, ok := complexvalue.NegateSymbol(carrier)
	if !ok {
		return api.ExpressionEmission{}, true,
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "complex negation has no runtime symbol",
			}
	}
	target, err := complexvalue.Call(
		context,
		symbol,
		[]tsgo.Expression{operand.Value()},
		operand.Requests()...,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result, err := api.NewExpressionEmission(
		operand.Before(),
		target.Value(),
		target.Requests(),
	)
	return result, true, err
}

func integerOperation(
	context api.Context,
	operator token.Token,
	carrier integervalue.Carrier,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if !integervalue.SupportsUnary(
		context.IntegerRepresentation(),
		carrier,
		operator,
	) {
		return api.ExpressionEmission{}, false, nil
	}
	target := operand.Value()
	switch operator {
	case token.ADD:
		if !integervalue.UsesBigInt(context.IntegerRepresentation(), carrier) {
			target = context.Factory().PrefixUnaryExpression(
				tsgo.PrefixUnaryExpressionOperatorKindPlusToken,
				target,
			)
		}
	case token.SUB:
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			target,
		)
	case token.XOR:
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindTildeToken,
			target,
		)
		var err error
		target, err = normalizeComplement(context, carrier, target)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	default:
		return api.ExpressionEmission{}, false, nil
	}
	result, err := api.NewExpressionEmission(
		operand.Before(),
		target,
		operand.Requests(),
	)
	return result, true, err
}

func normalizeComplement(
	context api.Context,
	carrier integervalue.Carrier,
	target tsgo.Expression,
) (tsgo.Expression, error) {
	switch {
	case integervalue.RequiresUint32Normalization(
		context.IntegerRepresentation(),
		carrier,
	):
		zero, err := integervalue.CarrierLiteral(context, carrier, "0")
		if err != nil {
			return nil, err
		}
		return context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken,
			),
			zero,
		), nil
	case !integervalue.UsesBigInt(context.IntegerRepresentation(), carrier) &&
		carrier.Width() > 32:
		return target, nil
	case !carrier.Signed():
		mask, ok := integervalue.UnsignedMask(carrier)
		if !ok {
			return nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "unsigned complement has no width mask",
			}
		}
		maskLiteral, err := integervalue.CarrierLiteral(context, carrier, mask)
		if err != nil {
			return nil, err
		}
		return context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorAmpersandToken,
			),
			maskLiteral,
		), nil
	default:
		return target, nil
	}
}

func floatOperation(
	context api.Context,
	operator token.Token,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if operator == token.ADD {
		result, err := api.NewExpressionEmission(
			operand.Before(),
			operand.Value(),
			operand.Requests(),
		)
		return result, true, err
	}
	if operator != token.SUB {
		return api.ExpressionEmission{}, false, nil
	}
	result, err := api.NewExpressionEmission(
		operand.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			operand.Value(),
		),
		operand.Requests(),
	)
	return result, true, err
}
