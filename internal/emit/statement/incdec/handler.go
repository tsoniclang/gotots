package incdec

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IncDecStmt,
) (api.StatementEmission, error) {
	expression, err := EmitExpression(context, children, source)
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
	children api.ChildEmitter,
	source *ast.IncDecStmt,
) (api.ExpressionEmission, error) {
	target, err := children.StoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !basictype.SupportsInteger(context.TypesSizes(), target.SourceType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
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
			target.Value(),
			operator,
		),
		target.Requests()...,
	), nil
}
