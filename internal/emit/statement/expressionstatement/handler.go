package expressionstatement

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ExprStmt,
) (api.StatementEmission, error) {
	target, err := EmitExpression(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := target.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(target.Value()),
	)
	return api.NewStatementEmission(statements, target.Requests())
}

func EmitExpression(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ExprStmt,
) (api.ExpressionEmission, error) {
	call, ok := source.X.(*ast.CallExpr)
	if ok {
		return children.DiscardedCall(
			context.WithRole(api.RoleExpressionStatement),
			call,
		)
	}
	receive, ok := source.X.(*ast.UnaryExpr)
	if !ok || receive.Op != token.ARROW {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	sourceType := context.TypesInfo().TypeOf(receive)
	if sourceType == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, receive)
	}
	return children.Expression(
		context.
			WithRole(api.RoleExpressionStatement).
			WithExpectedType(sourceType),
		receive,
	)
}
