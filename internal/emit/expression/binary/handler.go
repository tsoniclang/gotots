package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/expression/mapcomparison"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, error) {
	if target, handled, err := mapcomparison.Emit(
		context,
		children,
		source,
	); handled {
		return target, err
	}
	if source.Op == token.EQL || source.Op == token.NEQ {
		if target, ok, err := emitValueEquality(context, children, source); ok || err != nil {
			return target, err
		}
	}
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
	target := tsgo.Expression(context.Factory().BinaryExpression(
		nil,
		left.Value(),
		nil,
		operator,
		right.Value(),
	))
	return api.NewExpressionEmission(
		nil,
		target,
		api.CombineRequests(left.Requests(), right.Requests()),
	)
}

func emitValueEquality(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	if leftType == nil ||
		rightType == nil ||
		!types.Identical(leftType, rightType) ||
		!context.Values().RequiresCustomEquality(leftType) {
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
	equal, err := context.Values().Equal(
		context,
		source,
		leftType,
		left.Value(),
		right.Value(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	value := equal.Value()
	if source.Op == token.NEQ {
		value = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			value,
		)
	}
	return api.DirectExpression(
		value,
		api.CombineRequests(
			left.Requests(),
			right.Requests(),
			equal.Requests(),
		)...,
	), true, nil
}

func operationFor(
	context api.Context,
	source *ast.BinaryExpr,
) (tsgo.BinaryOperatorToken, types.Type, bool) {
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	integerType, integerOperands := integerOperandType(
		context.TypesSizes(),
		leftType,
		rightType,
	)
	switch {
	case isSignedArithmetic(source.Op) &&
		basictype.SupportsInteger(
			context.TypesSizes(),
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

func integerOperandType(
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
