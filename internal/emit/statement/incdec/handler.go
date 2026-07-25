package incdec

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	source *ast.IncDecStmt,
) (api.StatementEmission, error) {
	expression, err := EmitExpression(context, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := expression.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(expression.Value()),
	)
	return api.NewStatementEmission(statements, expression.Requests())
}

func EmitExpression(
	context api.Context,
	source *ast.IncDecStmt,
) (api.ExpressionEmission, error) {
	identifier, ok := source.X.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Uses[identifier].(*types.Var)
	if !ok || !isSupportedInteger(object.Type()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var operator tsgo.PostfixUnaryExpressionOperatorKind
	switch source.Tok {
	case token.INC:
		operator = tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken
	case token.DEC:
		operator = tsgo.PostfixUnaryExpressionOperatorKindMinusMinusToken
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return api.DirectExpression(
		context.Factory().PostfixUnaryExpression(
			context.Factory().Identifier(reference.Name()),
			operator,
		),
		reference.Requests()...,
	), nil
}

func isSupportedInteger(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return false
	}
	return basic.Kind() == types.Int || basic.Kind() == types.Int64
}
