package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, error) {
	operator, operandType, ok := operationFor(context, source)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	left, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	right, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(operandType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(left.Before()) != 0 || len(right.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.NewExpressionEmission(
		nil,
		context.Factory().BinaryExpression(
			nil,
			left.Value(),
			nil,
			operator,
			right.Value(),
		),
		api.CombineRequests(left.Requests(), right.Requests()),
	)
}

func operationFor(
	context api.Context,
	source *ast.BinaryExpr,
) (tsgo.BinaryOperatorToken, types.Type, bool) {
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	integerType, integerOperands := integerOperandType(leftType, rightType)
	switch {
	case isSignedArithmetic(source.Op) &&
		isSupportedSignedArithmetic(
			context,
			context.TypesInfo().TypeOf(source),
		):
		operandType := context.TypesInfo().TypeOf(source)
		if !types.AssignableTo(leftType, operandType) ||
			!types.AssignableTo(rightType, operandType) {
			return nil, nil, false
		}
		operator, ok := arithmeticOperator(context, source.Op)
		return operator, operandType, ok
	case isIntegerComparison(source.Op) && integerOperands:
		operator, ok := comparisonOperator(context, source.Op)
		return operator, integerType, ok
	case isLogicalOperator(source.Op) &&
		isSupportedBoolean(context.TypesInfo().TypeOf(source)) &&
		types.AssignableTo(leftType, types.Typ[types.Bool]) &&
		types.AssignableTo(rightType, types.Typ[types.Bool]):
		operator, ok := logicalOperator(context, source.Op)
		return operator, types.Typ[types.Bool], ok
	case source.Op == token.EQL &&
		types.AssignableTo(leftType, types.Typ[types.Bool]) &&
		types.AssignableTo(rightType, types.Typ[types.Bool]):
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		), types.Typ[types.Bool], true
	case source.Op == token.NEQ &&
		types.AssignableTo(leftType, types.Typ[types.Bool]) &&
		types.AssignableTo(rightType, types.Typ[types.Bool]):
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		), types.Typ[types.Bool], true
	default:
		return nil, nil, false
	}
}

func isSignedArithmetic(operator token.Token) bool {
	switch operator {
	case token.ADD, token.SUB, token.MUL:
		return true
	default:
		return false
	}
}

func arithmeticOperator(
	context api.Context,
	operator token.Token,
) (tsgo.BinaryOperatorToken, bool) {
	var target tsgo.BinaryOperator
	switch operator {
	case token.ADD:
		target = tsgo.BinaryOperatorPlusToken
	case token.SUB:
		target = tsgo.BinaryOperatorMinusToken
	case token.MUL:
		target = tsgo.BinaryOperatorAsteriskToken
	default:
		return nil, false
	}
	return context.Factory().BinaryOperatorToken(target), true
}

func isLogicalOperator(operator token.Token) bool {
	return operator == token.LAND || operator == token.LOR
}

func logicalOperator(
	context api.Context,
	operator token.Token,
) (tsgo.BinaryOperatorToken, bool) {
	switch operator {
	case token.LAND:
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorAmpersandAmpersandToken,
		), true
	case token.LOR:
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorBarBarToken,
		), true
	default:
		return nil, false
	}
}

func isSupportedBoolean(value types.Type) bool {
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && basic.Kind() == types.Bool
}

func isSupportedSignedArithmetic(context api.Context, value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := types.Unalias(value).(*types.Basic)
	if !ok {
		return false
	}
	switch basic.Kind() {
	case types.Int64:
		return true
	case types.Int:
		return context.TypesSizes().Sizeof(types.Typ[types.Int]) == 8
	default:
		return false
	}
}

func isSupportedSignedInteger(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && (basic.Kind() == types.Int || basic.Kind() == types.Int64)
}

func integerOperandType(left, right types.Type) (types.Type, bool) {
	for _, candidate := range []types.Type{left, right} {
		if !isSupportedSignedInteger(candidate) {
			continue
		}
		if types.AssignableTo(left, candidate) && types.AssignableTo(right, candidate) {
			return candidate, true
		}
	}
	return nil, false
}

func isIntegerComparison(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func comparisonOperator(
	context api.Context,
	operator token.Token,
) (tsgo.BinaryOperatorToken, bool) {
	var target tsgo.BinaryOperator
	switch operator {
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
