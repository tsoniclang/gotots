package goroutine

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.GoStmt,
) (api.StatementEmission, error) {
	if source == nil || source.Call == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	call, err := children.DiscardedCall(
		context.WithRole(api.RoleGoroutineCall),
		source.Call,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := call.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(call.Value()),
	)
	return api.NewStatementEmission(statements, call.Requests())
}
