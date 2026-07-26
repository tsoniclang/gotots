package expressionstatement

import (
	"go/ast"

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
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	target, err := children.DiscardedCall(
		context.WithRole(api.RoleExpressionStatement),
		call,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return target, nil
}
