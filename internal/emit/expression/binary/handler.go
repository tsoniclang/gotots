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
) (tsgo.Expression, error) {
	operator, ok := operatorFor(context, source)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	left, err := children.Expression(context.WithRole(api.RoleBinaryLeft), source.X)
	if err != nil {
		return nil, err
	}
	right, err := children.Expression(context.WithRole(api.RoleBinaryRight), source.Y)
	if err != nil {
		return nil, err
	}
	return context.Factory().BinaryExpression(
		nil,
		left,
		nil,
		operator,
		right,
	), nil
}

func operatorFor(
	context api.Context,
	source *ast.BinaryExpr,
) (tsgo.BinaryOperatorToken, bool) {
	switch {
	case source.Op == token.ADD && isInt(context.TypesInfo().TypeOf(source)):
		return context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorPlusToken), true
	case source.Op == token.EQL &&
		isBool(context.TypesInfo().TypeOf(source.X)) &&
		isBool(context.TypesInfo().TypeOf(source.Y)):
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		), true
	case source.Op == token.NEQ &&
		isBool(context.TypesInfo().TypeOf(source.X)) &&
		isBool(context.TypesInfo().TypeOf(source.Y)):
		return context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		), true
	default:
		return nil, false
	}
}

func isInt(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && basic.Kind() == types.Int
}

func isBool(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && basic.Info()&types.IsBoolean != 0
}
