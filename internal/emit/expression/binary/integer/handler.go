package integer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
	if len(left.Before()) != 0 || len(right.Before()) != 0 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, ok := target(
		context,
		source.Op,
		carrier,
		left.Value(),
		right.Value(),
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	return api.DirectExpression(
		target,
		api.CombineRequests(left.Requests(), right.Requests())...,
	), true, nil
}

func OperationFor(
	context api.Context,
	source *ast.BinaryExpr,
) (tsgo.BinaryOperatorToken, types.Type, bool) {
	leftType, rightType, carrier, ok := operationTypes(context, source)
	if !ok ||
		!types.Identical(leftType, rightType) ||
		source.Op == token.AND_NOT ||
		needsUint32Normalization(context, carrier, source.Op) {
		return nil, nil, false
	}
	operator, represented := targetOperator(context, source.Op, carrier)
	return operator, leftType, represented
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
		typeAndValue, ok := context.TypesInfo().Types[source.Y]
		if !ok ||
			!integervalue.SupportsShift(
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
		); !represented {
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
		zero, err := api.IntegerLiteral(
			context.Factory(),
			api.IntegerRepresentationNumber,
			"0",
		)
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
	case token.QUO:
		target = tsgo.BinaryOperatorSlashToken
	case token.REM:
		target = tsgo.BinaryOperatorPercentToken
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
