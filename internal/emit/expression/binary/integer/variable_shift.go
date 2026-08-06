package integer

import (
	"fmt"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ApplyVariableShift(
	context api.Context,
	operator token.Token,
	carrier integervalue.Carrier,
	countCarrier integervalue.Carrier,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if !integervalue.SupportsVariableShift(
		context.IntegerRepresentation(),
		carrier,
		operator,
	) {
		return api.ExpressionEmission{}, false, nil
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var target tsgo.Expression
	var normalizationRequests []api.RootRequest
	carrierRepresentation, represented := integervalue.CarrierRepresentation(
		context.IntegerRepresentation(),
		carrier,
	)
	if !represented {
		return api.ExpressionEmission{}, false, nil
	}
	switch carrierRepresentation {
	case api.IntegerCarrierNumber:
		target, normalizationRequests, err = numberVariableShift(
			context,
			panicReference.Name(),
			operator,
			carrier,
			countCarrier,
			left.Value(),
			right.Value(),
		)
	case api.IntegerCarrierBigInt:
		target, normalizationRequests, err = bigIntVariableShift(
			context,
			panicReference.Name(),
			operator,
			carrier,
			countCarrier,
			left.Value(),
			right.Value(),
		)
	default:
		return api.ExpressionEmission{}, false, nil
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return api.DirectExpression(
		target,
		api.CombineRequests(
			left.Requests(),
			right.Requests(),
			panicReference.Requests(),
			normalizationRequests,
		)...,
	), true, nil
}

func numberVariableShift(
	context api.Context,
	panicName string,
	operator token.Token,
	carrier integervalue.Carrier,
	countCarrier integervalue.Carrier,
	left tsgo.Expression,
	right tsgo.Expression,
) (tsgo.Expression, []api.RootRequest, error) {
	factory := context.Factory()
	countZero, err := integervalue.CarrierLiteral(context, countCarrier, "0")
	if err != nil {
		return nil, nil, err
	}
	countWidth, err := integervalue.CarrierLiteral(
		context,
		countCarrier,
		fmt.Sprintf("%d", carrier.Width()),
	)
	if err != nil {
		return nil, nil, err
	}
	shiftCount, represented := alignShiftCount(
		context,
		carrier,
		countCarrier,
		right,
	)
	if !represented {
		return nil, nil, &api.IntegerRepresentationError{
			Representation: context.IntegerRepresentation(),
		}
	}
	zero := tsgo.Expression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))
	normalized, err := integervalue.NormalizeFixedWidth(context, carrier, left)
	if err != nil {
		return nil, nil, err
	}
	shifted, err := integervalue.NormalizeFixedWidth(
		context,
		carrier,
		factory.BinaryExpression(
			nil,
			normalized.Value(),
			nil,
			shiftOperator(factory, operator, carrier.Signed()),
			shiftCount,
		),
	)
	if err != nil {
		return nil, nil, err
	}
	return guardedShift(
			context,
			panicName,
			right,
			countZero,
			countWidth,
			wideShiftResult(
				factory,
				operator,
				carrier.Signed(),
				normalized.Value(),
				zero,
			),
			shifted.Value(),
		), api.CombineRequests(
			normalized.Requests(),
			shifted.Requests(),
		), nil
}

func bigIntVariableShift(
	context api.Context,
	panicName string,
	operator token.Token,
	carrier integervalue.Carrier,
	countCarrier integervalue.Carrier,
	left tsgo.Expression,
	right tsgo.Expression,
) (tsgo.Expression, []api.RootRequest, error) {
	factory := context.Factory()
	countZero, err := integervalue.CarrierLiteral(context, countCarrier, "0")
	if err != nil {
		return nil, nil, err
	}
	countWidth, err := integervalue.CarrierLiteral(
		context,
		countCarrier,
		fmt.Sprintf("%d", carrier.Width()),
	)
	if err != nil {
		return nil, nil, err
	}
	shiftCount, represented := alignShiftCount(
		context,
		carrier,
		countCarrier,
		right,
	)
	if !represented {
		return nil, nil, &api.IntegerRepresentationError{
			Representation: context.IntegerRepresentation(),
		}
	}
	zero := tsgo.Expression(factory.BigIntLiteral("0n", tsgo.TokenFlagsNone))
	normalized, err := integervalue.NormalizeFixedWidth(context, carrier, left)
	if err != nil {
		return nil, nil, err
	}
	shifted, err := integervalue.NormalizeFixedWidth(
		context,
		carrier,
		factory.BinaryExpression(
			nil,
			normalized.Value(),
			nil,
			shiftOperator(factory, operator, true),
			shiftCount,
		),
	)
	if err != nil {
		return nil, nil, err
	}
	return guardedShift(
			context,
			panicName,
			right,
			countZero,
			countWidth,
			wideShiftResult(
				factory,
				operator,
				carrier.Signed(),
				normalized.Value(),
				zero,
			),
			shifted.Value(),
		), api.CombineRequests(
			normalized.Requests(),
			shifted.Requests(),
		), nil
}

func guardedShift(
	context api.Context,
	panicName string,
	count tsgo.Expression,
	zero tsgo.Expression,
	width tsgo.Expression,
	wide tsgo.Expression,
	shifted tsgo.Expression,
) tsgo.Expression {
	factory := context.Factory()
	return factory.ConditionalExpression(
		factory.BinaryExpression(
			nil,
			count,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorLessThanToken),
			zero,
		),
		factory.QuestionToken(),
		panicruntime.Call(
			factory,
			panicName,
			factory.StringLiteral(
				"negative shift amount",
				tsgo.TokenFlagsNone,
			),
		),
		factory.ColonToken(),
		factory.ConditionalExpression(
			factory.BinaryExpression(
				nil,
				count,
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorGreaterThanEqualsToken,
				),
				width,
			),
			factory.QuestionToken(),
			wide,
			factory.ColonToken(),
			shifted,
		),
	)
}

func wideShiftResult(
	factory tsgo.Factory,
	operator token.Token,
	signed bool,
	left tsgo.Expression,
	zero tsgo.Expression,
) tsgo.Expression {
	if operator != token.SHR || !signed {
		return zero
	}
	return factory.ConditionalExpression(
		factory.BinaryExpression(
			nil,
			left,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorLessThanToken),
			zero,
		),
		factory.QuestionToken(),
		factory.PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			oneLike(factory, zero),
		),
		factory.ColonToken(),
		zero,
	)
}

func oneLike(factory tsgo.Factory, zero tsgo.Expression) tsgo.Expression {
	if _, bigInt := zero.(tsgo.BigIntLiteral); bigInt {
		return factory.BigIntLiteral("1n", tsgo.TokenFlagsNone)
	}
	return factory.NumericLiteral("1", tsgo.TokenFlagsNone)
}

func shiftOperator(
	factory tsgo.Factory,
	operator token.Token,
	signed bool,
) tsgo.BinaryOperatorToken {
	if operator == token.SHL {
		return factory.BinaryOperatorToken(
			tsgo.BinaryOperatorLessThanLessThanToken,
		)
	}
	if signed {
		return factory.BinaryOperatorToken(
			tsgo.BinaryOperatorGreaterThanGreaterThanToken,
		)
	}
	return factory.BinaryOperatorToken(
		tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken,
	)
}
