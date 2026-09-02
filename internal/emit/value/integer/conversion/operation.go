package integerconversion

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Convert(
	context api.Context,
	sourceABI api.ScalarABI,
	targetABI api.ScalarABI,
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
	targetRepresentation, err := targetABI.Carrier(targetCarrier.Alias())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceRepresentation := api.IntegerCarrierInvalid
	if integerSource {
		sourceRepresentation, err = sourceABI.Carrier(sourceInteger.Alias())
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if integerSource &&
		sourceRepresentation == targetRepresentation &&
		integervalue.CanConvertDirectly(sourceInteger, targetCarrier) {
		return operand, nil
	}
	value := operand.Value()
	requests := operand.Requests()
	switch targetRepresentation {
	case api.IntegerCarrierNumber:
		if integerSource && sourceRepresentation == api.IntegerCarrierBigInt {
			value = normalizeBigInt(context, targetCarrier, value)
			value = context.Factory().CallExpression(
				api.TargetIntrinsicNumber.Expression(context.Factory()),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			)
		} else if targetCarrier.Width() <= 32 {
			if floatSource {
				value = mathCall(context, "trunc", value)
			}
			value = integervalue.NormalizeNumber(
				context.Factory(),
				targetCarrier,
				value,
			)
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
	case api.IntegerCarrierBigInt:
		if floatSource || sourceRepresentation == api.IntegerCarrierNumber {
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
				Representation: targetABI.IntegerRepresentation(),
			}
	}
	return api.NewExpressionEmission(
		operand.Before(),
		value,
		requests,
	)
}

func ConvertRepresentation(
	context api.Context,
	carrier integervalue.Carrier,
	source api.IntegerCarrier,
	target api.IntegerCarrier,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !source.Valid() || !target.Valid() {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "integer carrier boundary is invalid",
		}
	}
	if source == target {
		return operand, nil
	}
	value := operand.Value()
	requests := operand.Requests()
	var err error
	switch {
	case source == api.IntegerCarrierNumber &&
		target == api.IntegerCarrierBigInt:
		value, requests, err = numberToBigInt(
			context,
			value,
			requests,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value = normalizeBigInt(context, carrier, value)
	case source == api.IntegerCarrierBigInt &&
		target == api.IntegerCarrierNumber:
		value = normalizeBigInt(context, carrier, value)
		value = context.Factory().CallExpression(
			api.TargetIntrinsicNumber.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		)
	default:
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "integer carrier boundary is unsupported",
		}
	}
	return api.NewExpressionEmission(
		operand.Before(),
		value,
		requests,
	)
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
