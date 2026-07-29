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
	switch context.IntegerRepresentation() {
	case api.IntegerRepresentationNumber:
		target = numberVariableShift(
			context,
			panicReference.Name(),
			operator,
			carrier,
			left.Value(),
			right.Value(),
		)
	case api.IntegerRepresentationBigInt:
		target = bigIntVariableShift(
			context,
			panicReference.Name(),
			operator,
			carrier,
			left.Value(),
			right.Value(),
		)
	default:
		return api.ExpressionEmission{}, false, nil
	}
	return api.DirectExpression(
		target,
		api.CombineRequests(
			left.Requests(),
			right.Requests(),
			panicReference.Requests(),
		)...,
	), true, nil
}

func numberVariableShift(
	context api.Context,
	panicName string,
	operator token.Token,
	carrier integervalue.Carrier,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.Expression {
	factory := context.Factory()
	zero := tsgo.Expression(
		factory.NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	normalized := normalizeNumber(factory, carrier, left)
	shifted := normalizeNumber(
		factory,
		carrier,
		factory.BinaryExpression(
			nil,
			normalized,
			nil,
			shiftOperator(factory, operator, carrier.Signed()),
			right,
		),
	)
	return guardedShift(
		context,
		panicName,
		right,
		zero,
		factory.NumericLiteral(
			fmt.Sprintf("%d", carrier.Width()),
			tsgo.TokenFlagsNone,
		),
		wideShiftResult(factory, operator, carrier.Signed(), normalized, zero),
		shifted,
	)
}

func bigIntVariableShift(
	context api.Context,
	panicName string,
	operator token.Token,
	carrier integervalue.Carrier,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.Expression {
	factory := context.Factory()
	zero := tsgo.Expression(
		factory.BigIntLiteral("0n", tsgo.TokenFlagsNone),
	)
	normalized := normalizeBigInt(factory, carrier, left)
	shifted := normalizeBigInt(
		factory,
		carrier,
		factory.BinaryExpression(
			nil,
			normalized,
			nil,
			shiftOperator(factory, operator, true),
			right,
		),
	)
	return guardedShift(
		context,
		panicName,
		right,
		zero,
		factory.BigIntLiteral(
			fmt.Sprintf("%dn", carrier.Width()),
			tsgo.TokenFlagsNone,
		),
		wideShiftResult(factory, operator, carrier.Signed(), normalized, zero),
		shifted,
	)
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

func normalizeNumber(
	factory tsgo.Factory,
	carrier integervalue.Carrier,
	value tsgo.Expression,
) tsgo.Expression {
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
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
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

func normalizeBigInt(
	factory tsgo.Factory,
	carrier integervalue.Carrier,
	value tsgo.Expression,
) tsgo.Expression {
	member := "asUintN"
	if carrier.Signed() {
		member = "asIntN"
	}
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("BigInt"),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			factory.NumericLiteral(
				fmt.Sprintf("%d", carrier.Width()),
				tsgo.TokenFlagsNone,
			),
			value,
		},
		tsgo.NodeFlagsNone,
	)
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
