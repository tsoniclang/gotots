package deferstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	callexpression "github.com/tsoniclang/gotots/internal/emit/expression/call"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.DeferStmt,
) (api.StatementEmission, error) {
	if source == nil || source.Call == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	control, selected := context.DeferControl()
	request, err := context.CallableControlRequest(
		api.CallableControlDefer,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if !selected {
		return api.NewStatementEmission(nil, []api.RootRequest{request})
	}
	deferred, err := callexpression.EmitDeferred(
		context.WithRole(api.RoleExpressionStatement),
		children,
		source.Call,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := deferred.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(control.Stack()),
					nil,
					context.Factory().Identifier("push"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{deferred.Value()},
				tsgo.NodeFlagsNone,
			),
		),
	)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(deferred.Requests(), []api.RootRequest{request}),
	)
}
