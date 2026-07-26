package incdec

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
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
	if !ok ||
		!basictype.SupportsExactInt32(context.TypesSizes(), object.Type()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var operator tsgo.BinaryOperator
	switch source.Tok {
	case token.INC:
		operator = tsgo.BinaryOperatorPlusToken
	case token.DEC:
		operator = tsgo.BinaryOperatorMinusToken
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	value := context.Factory().BinaryExpression(
		nil,
		context.Factory().Identifier(reference.Name()),
		nil,
		context.Factory().BinaryOperatorToken(operator),
		context.Factory().NumericLiteral("1", tsgo.TokenFlagsNone),
	)
	wrapped := context.Factory().BinaryExpression(
		nil,
		context.Factory().ParenthesizedExpression(value),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorBarToken),
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	return api.DirectExpression(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().Identifier(reference.Name()),
			nil,
			context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			wrapped,
		),
		reference.Requests()...,
	), nil
}
