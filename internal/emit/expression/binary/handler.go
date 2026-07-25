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
	case source.Op == token.ADD && isSupportedSignedInteger(context.TypesInfo().TypeOf(source)):
		operandType := context.TypesInfo().TypeOf(source)
		if !types.AssignableTo(leftType, operandType) ||
			!types.AssignableTo(rightType, operandType) {
			return nil, nil, false
		}
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorPlusToken,
		), operandType, true
	case isIntegerComparison(source.Op) && integerOperands:
		operator, ok := comparisonOperator(context, source.Op)
		return operator, integerType, ok
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
