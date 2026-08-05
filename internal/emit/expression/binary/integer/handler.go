package integer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	leftType, rightType, carrier, ok := operationTypes(context, source)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	left, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(leftType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(rightType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err = projectDefinedShiftCount(context, source, right)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	operands, err := expressionoperands.PreservePair(
		context,
		left,
		right,
		api.TemporaryBinaryOperand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	facts, hasFacts := context.TypesInfo().TypeAndValue(source.Y)
	rightCarrier := carrier
	if source.Op == token.SHL || source.Op == token.SHR {
		var rightOK bool
		rightCarrier, rightOK = shiftCountCarrier(
			context.TypesSizes(),
			rightType,
		)
		if !rightOK {
			return api.ExpressionEmission{}, false, nil
		}
	}
	var target api.ExpressionEmission
	var handled bool
	if (source.Op == token.SHL || source.Op == token.SHR) &&
		hasFacts &&
		facts.Value == nil {
		target, handled, err = ApplyVariableShift(
			context,
			source.Op,
			carrier,
			rightCarrier,
			operands.Left(),
			operands.Right(),
		)
	} else {
		target, handled, err = Apply(
			context,
			source.Op,
			carrier,
			rightCarrier,
			operands.Left(),
			operands.Right(),
		)
	}
	if err != nil || !handled {
		return target, handled, err
	}
	target, err = expressionoperands.Finish(operands, target)
	return target, true, err
}

func Apply(
	context api.Context,
	operator token.Token,
	carrier integervalue.Carrier,
	rightCarrier integervalue.Carrier,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if symbol, ok := runtimeOperation(
		context.IntegerRepresentation(),
		carrier,
		operator,
	); ok {
		result, err := callRuntime(context, symbol, left, right)
		return result, true, err
	}
	rightValue := right.Value()
	if operator == token.SHL || operator == token.SHR {
		var represented bool
		rightValue, represented = alignShiftCount(
			context,
			carrier,
			rightCarrier,
			rightValue,
		)
		if !represented {
			return api.ExpressionEmission{}, false, nil
		}
	}
	target, ok := target(
		context,
		operator,
		carrier,
		left.Value(),
		rightValue,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	return api.DirectExpression(
		target,
		api.CombineRequests(left.Requests(), right.Requests())...,
	), true, nil
}

func alignShiftCount(
	context api.Context,
	left integervalue.Carrier,
	right integervalue.Carrier,
	value tsgo.Expression,
) (tsgo.Expression, bool) {
	leftRepresentation, leftOK := integervalue.CarrierRepresentation(
		context.IntegerRepresentation(),
		left,
	)
	rightRepresentation, rightOK := integervalue.CarrierRepresentation(
		context.IntegerRepresentation(),
		right,
	)
	if !leftOK || !rightOK {
		return nil, false
	}
	if leftRepresentation == rightRepresentation {
		return value, true
	}
	intrinsic := api.TargetIntrinsicNumber
	if leftRepresentation == api.IntegerCarrierBigInt {
		intrinsic = api.TargetIntrinsicBigInt
	}
	return context.Factory().CallExpression(
		intrinsic.Expression(context.Factory()),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	), true
}

func runtimeOperation(
	representation api.IntegerRepresentation,
	carrier integervalue.Carrier,
	operator token.Token,
) (api.RuntimeSymbol, bool) {
	carrierRepresentation, ok := integervalue.CarrierRepresentation(
		representation,
		carrier,
	)
	if !ok {
		return api.RuntimeInvalid, false
	}
	switch carrierRepresentation {
	case api.IntegerCarrierBigInt:
		switch operator {
		case token.QUO:
			return api.RuntimeIntegerDivide, true
		case token.REM:
			return api.RuntimeIntegerRemainder, true
		}
	case api.IntegerCarrierNumber:
		switch operator {
		case token.QUO:
			return api.RuntimeNumberIntDivide, true
		case token.REM:
			return api.RuntimeNumberIntRemainder, true
		}
	default:
	}
	return api.RuntimeInvalid, false
}

func callRuntime(
	context api.Context,
	symbol api.RuntimeSymbol,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().Runtime(
		symbol,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{left.Value(), right.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			left.Requests(),
			right.Requests(),
			reference.Requests(),
		)...,
	), nil
}

func operationTypes(
	context api.Context,
	source *ast.BinaryExpr,
) (types.Type, types.Type, integervalue.Carrier, bool) {
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	resultType := context.TypesInfo().TypeOf(source)
	if isComparison(source.Op) {
		operandType, ok := operandType(
			context.TypesSizes(),
			leftType,
			rightType,
		)
		if !ok {
			return nil, nil, integervalue.Carrier{}, false
		}
		carrier, ok := integervalue.Describe(context.TypesSizes(), operandType)
		if !ok {
			return nil, nil, integervalue.Carrier{}, false
		}
		return operandType, operandType, carrier, true
	}
	carrier, ok := integervalue.Describe(context.TypesSizes(), resultType)
	if !ok || !types.AssignableTo(leftType, resultType) {
		return nil, nil, integervalue.Carrier{}, false
	}
	switch {
	case integervalue.SupportsArithmetic(context.IntegerRepresentation(), source.Op),
		integervalue.SupportsBitwise(context.IntegerRepresentation(), carrier, source.Op):
		if !types.AssignableTo(rightType, resultType) {
			return nil, nil, integervalue.Carrier{}, false
		}
		return resultType, resultType, carrier, true
	case source.Op == token.SHL || source.Op == token.SHR:
		typeAndValue, ok := context.TypesInfo().TypeAndValue(source.Y)
		if !ok {
			return nil, nil, integervalue.Carrier{}, false
		}
		definedCount := isDefinedIntegerShiftCount(
			context.TypesSizes(),
			rightType,
		)
		if typeAndValue.Value == nil {
			if _, rightOK := integervalue.Describe(
				context.TypesSizes(),
				rightType,
			); !rightOK &&
				!definedCount ||
				!integervalue.SupportsVariableShift(
					context.IntegerRepresentation(),
					carrier,
					source.Op,
				) {
				return nil, nil, integervalue.Carrier{}, false
			}
		} else if !integervalue.SupportsShift(
			context.IntegerRepresentation(),
			carrier,
			source.Op,
			typeAndValue.Value,
		) {
			return nil, nil, integervalue.Carrier{}, false
		}
		expectedRight := rightType
		if _, represented := integervalue.Describe(
			context.TypesSizes(),
			rightType,
		); !represented && !definedCount {
			expectedRight = resultType
		}
		if !types.AssignableTo(rightType, expectedRight) {
			return nil, nil, integervalue.Carrier{}, false
		}
		return resultType, expectedRight, carrier, true
	default:
		return nil, nil, integervalue.Carrier{}, false
	}
}

func target(
	context api.Context,
	operator token.Token,
	carrier integervalue.Carrier,
	left tsgo.Expression,
	right tsgo.Expression,
) (tsgo.Expression, bool) {
	if operator == token.AND_NOT {
		right = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindTildeToken,
			right,
		)
		operator = token.AND
	}
	targetOperator, ok := targetOperator(context, operator, carrier)
	if !ok {
		return nil, false
	}
	target := tsgo.Expression(context.Factory().BinaryExpression(
		nil,
		left,
		nil,
		targetOperator,
		right,
	))
	if needsUint32Normalization(context, carrier, operator) {
		zero, err := integervalue.CarrierLiteral(context, carrier, "0")
		if err != nil {
			return nil, false
		}
		target = context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken,
			),
			zero,
		)
	}
	return target, true
}

func needsUint32Normalization(
	context api.Context,
	carrier integervalue.Carrier,
	operator token.Token,
) bool {
	if !integervalue.RequiresUint32Normalization(
		context.IntegerRepresentation(),
		carrier,
	) {
		return false
	}
	switch operator {
	case token.AND, token.OR, token.XOR, token.SHL:
		return true
	default:
		return false
	}
}

func targetOperator(
	context api.Context,
	operator token.Token,
	carrier integervalue.Carrier,
) (tsgo.BinaryOperatorToken, bool) {
	var target tsgo.BinaryOperator
	switch operator {
	case token.ADD:
		target = tsgo.BinaryOperatorPlusToken
	case token.SUB:
		target = tsgo.BinaryOperatorMinusToken
	case token.MUL:
		target = tsgo.BinaryOperatorAsteriskToken
	case token.AND:
		target = tsgo.BinaryOperatorAmpersandToken
	case token.OR:
		target = tsgo.BinaryOperatorBarToken
	case token.XOR:
		target = tsgo.BinaryOperatorCaretToken
	case token.SHL:
		target = tsgo.BinaryOperatorLessThanLessThanToken
	case token.SHR:
		if integervalue.RequiresUint32Normalization(
			context.IntegerRepresentation(),
			carrier,
		) {
			target = tsgo.BinaryOperatorGreaterThanGreaterThanGreaterThanToken
		} else {
			target = tsgo.BinaryOperatorGreaterThanGreaterThanToken
		}
	case token.EQL:
		target = tsgo.BinaryOperatorEqualsEqualsEqualsToken
	case token.NEQ:
		target = tsgo.BinaryOperatorExclamationEqualsEqualsToken
	case token.LSS:
		target = tsgo.BinaryOperatorLessThanToken
	case token.LEQ:
		target = tsgo.BinaryOperatorLessThanEqualsToken
	case token.GTR:
		target = tsgo.BinaryOperatorGreaterThanToken
	case token.GEQ:
		target = tsgo.BinaryOperatorGreaterThanEqualsToken
	default:
		return nil, false
	}
	return context.Factory().BinaryOperatorToken(target), true
}

func operandType(
	sizes types.Sizes,
	left types.Type,
	right types.Type,
) (types.Type, bool) {
	for _, candidate := range []types.Type{left, right} {
		if !basictype.SupportsInteger(sizes, candidate) {
			continue
		}
		if types.AssignableTo(left, candidate) && types.AssignableTo(right, candidate) {
			return candidate, true
		}
	}
	return nil, false
}

func isComparison(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}
