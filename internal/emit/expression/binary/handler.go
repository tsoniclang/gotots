package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
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
		if target, ok, err := emitSliceNilEquality(
			context,
			children,
			source,
		); ok || err != nil {
			return target, err
		}
		if target, ok, err := emitValueEquality(context, children, source); ok || err != nil {
			return target, err
		}
	}
	if target, ok, err := integerbinary.Emit(
		context,
		children,
		source,
	); ok || err != nil {
		return target, err
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
	if leftType == nil || rightType == nil {
		return api.ExpressionEmission{}, false, nil
	}
	var operandType types.Type
	for _, candidate := range []types.Type{leftType, rightType} {
		if context.Values().RequiresCustomEquality(context, candidate) &&
			types.AssignableTo(leftType, candidate) &&
			types.AssignableTo(rightType, candidate) {
			operandType = candidate
			break
		}
	}
	if operandType == nil {
		return api.ExpressionEmission{}, false, nil
	}
	left, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(operandType),
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
		operandType,
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
	switch {
	case source.Op == token.ADD &&
		basictype.SupportsString(context.TypesInfo().TypeOf(source)) &&
		types.AssignableTo(leftType, context.TypesInfo().TypeOf(source)) &&
		types.AssignableTo(rightType, context.TypesInfo().TypeOf(source)):
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorPlusToken,
		), context.TypesInfo().TypeOf(source), true
	case isStringComparison(source.Op) &&
		basictype.SupportsString(leftType) &&
		basictype.SupportsString(rightType):
		operator, ok := stringOperator(context, source.Op)
		return operator, types.Typ[types.String], ok
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

func isStringComparison(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func stringOperator(
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
