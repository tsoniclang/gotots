package incdec

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, source *ast.IncDecStmt) (tsgo.Statement, error) {
	expression, err := EmitExpression(context, source)
	if err != nil {
		return nil, err
	}
	return context.Factory().ExpressionStatement(expression), nil
}

func EmitExpression(
	context api.Context,
	source *ast.IncDecStmt,
) (tsgo.Expression, error) {
	identifier, ok := source.X.(*ast.Ident)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Uses[identifier].(*types.Var)
	if !ok || !isSupportedInteger(object.Type()) {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	name, err := context.Names().Reference(object)
	if err != nil {
		return nil, err
	}
	var operator tsgo.PostfixUnaryExpressionOperatorKind
	switch source.Tok {
	case token.INC:
		operator = tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken
	case token.DEC:
		operator = tsgo.PostfixUnaryExpressionOperatorKindMinusMinusToken
	default:
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	return context.Factory().PostfixUnaryExpression(
		context.Factory().Identifier(name),
		operator,
	), nil
}

func isSupportedInteger(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return false
	}
	return basic.Kind() == types.Int || basic.Kind() == types.Int64
}
