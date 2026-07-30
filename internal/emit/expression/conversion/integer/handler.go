package integer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
) (api.ExpressionEmission, error) {
	if source == nil || len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(sourceType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return Convert(
		context,
		source,
		sourceType,
		targetType,
		operand,
	)
}

func Convert(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	targetCarrier, ok := integervalue.Describe(
		context.TypesSizes(),
		targetType,
	)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceInteger, integerSource := integervalue.Describe(
		context.TypesSizes(),
		sourceType,
	)
	_, floatSource := floatvalue.Describe(sourceType)
	if !integerSource && !floatSource {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if integerSource &&
		integervalue.CanConvertDirectly(sourceInteger, targetCarrier) {
		return operand, nil
	}
	value := operand.Value()
	requests := operand.Requests()
	var err error
	switch context.IntegerRepresentation() {
	case api.IntegerRepresentationNumber:
		if targetCarrier.Width() <= 32 {
			if floatSource {
				value = mathCall(context, "trunc", value)
			}
			value = normalizeNumber(context, targetCarrier, value)
		} else {
			value, requests, err = normalizeNumber64(
				context,
				targetCarrier,
				value,
				requests,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
	case api.IntegerRepresentationBigInt:
		if floatSource {
			value, requests, err = numberToBigInt(
				context,
				operand.Value(),
				requests,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
		value = normalizeBigInt(context, targetCarrier, value)
	default:
		return api.ExpressionEmission{},
			&api.IntegerRepresentationError{
				Representation: context.IntegerRepresentation(),
			}
	}
	return api.NewExpressionEmission(
		operand.Before(),
		value,
		requests,
	)
}

func normalizeNumber(
	context api.Context,
	target integervalue.Carrier,
	value tsgo.Expression,
) tsgo.Expression {
	factory := context.Factory()
	zero := factory.NumericLiteral("0", tsgo.TokenFlagsNone)
	switch {
	case target.Signed() && target.Width() == 8:
		return binary(
			factory,
			binary(
				factory,
				value,
				tsgo.BinaryOperatorLessThanLessThanToken,
				factory.NumericLiteral("24", tsgo.TokenFlagsNone),
			),
			tsgo.BinaryOperatorGreaterThanGreaterThanToken,
			factory.NumericLiteral("24", tsgo.TokenFlagsNone),
		)
	case target.Signed() && target.Width() == 16:
		return binary(
			factory,
			binary(
				factory,
				value,
				tsgo.BinaryOperatorLessThanLessThanToken,
				factory.NumericLiteral("16", tsgo.TokenFlagsNone),
			),
			tsgo.BinaryOperatorGreaterThanGreaterThanToken,
			factory.NumericLiteral("16", tsgo.TokenFlagsNone),
		)
	case target.Signed() && target.Width() == 32:
		return binary(factory, value, tsgo.BinaryOperatorBarToken, zero)
	case !target.Signed() && target.Width() == 8:
		return binary(
			factory,
			value,
			tsgo.BinaryOperatorAmpersandToken,
			factory.NumericLiteral("255", tsgo.TokenFlagsNone),
		)
	case !target.Signed() && target.Width() == 16:
		return binary(
			factory,
			value,
			tsgo.BinaryOperatorAmpersandToken,
			factory.NumericLiteral("65535", tsgo.TokenFlagsNone),
		)
	case !target.Signed() && target.Width() == 32:
		return binary(
			factory,
			value,
			tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken,
			zero,
		)
	default:
		panic("number normalization target is not 8, 16, or 32 bits")
	}
}

func normalizeNumber64(
	context api.Context,
	target integervalue.Carrier,
	value tsgo.Expression,
	requests []api.RootRequest,
) (tsgo.Expression, []api.RootRequest, error) {
	asBigInt, requests, err := numberToBigInt(
		context,
		value,
		requests,
	)
	if err != nil {
		return nil, nil, err
	}
	normalized := normalizeBigInt(context, target, asBigInt)
	return context.Factory().CallExpression(
		api.TargetIntrinsicNumber.Expression(context.Factory()),
		nil,
		nil,
		[]tsgo.Expression{normalized},
		tsgo.NodeFlagsNone,
	), requests, nil
}

func numberToBigInt(
	context api.Context,
	value tsgo.Expression,
	requests []api.RootRequest,
) (tsgo.Expression, []api.RootRequest, error) {
	reference, err := context.Names().Runtime(
		api.RuntimeNumberToBigInt,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	return globalCall(context, reference.Name(), value),
		api.CombineRequests(requests, reference.Requests()),
		nil
}

func normalizeBigInt(
	context api.Context,
	target integervalue.Carrier,
	value tsgo.Expression,
) tsgo.Expression {
	member := "asUintN"
	if target.Signed() {
		member = "asIntN"
	}
	return propertyCall(
		context,
		"BigInt",
		member,
		context.Factory().NumericLiteral(
			integerWidth(target),
			tsgo.TokenFlagsNone,
		),
		value,
	)
}

func integerWidth(target integervalue.Carrier) string {
	switch target.Width() {
	case 8:
		return "8"
	case 16:
		return "16"
	case 32:
		return "32"
	case 64:
		return "64"
	default:
		panic("integer conversion target width is invalid")
	}
}

func mathCall(
	context api.Context,
	member string,
	value tsgo.Expression,
) tsgo.Expression {
	return propertyCall(context, "Math", member, value)
}

func propertyCall(
	context api.Context,
	receiver string,
	member string,
	arguments ...tsgo.Expression,
) tsgo.Expression {
	factory := context.Factory()
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(receiver),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func globalCall(
	context api.Context,
	name string,
	argument tsgo.Expression,
) tsgo.Expression {
	return context.Factory().CallExpression(
		context.Factory().Identifier(name),
		nil,
		nil,
		[]tsgo.Expression{argument},
		tsgo.NodeFlagsNone,
	)
}

func binary(
	factory tsgo.Factory,
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.Expression {
	return factory.BinaryExpression(
		nil,
		left,
		nil,
		factory.BinaryOperatorToken(operator),
		right,
	)
}
