package block

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BlockStmt,
) (api.BlockEmission, error) {
	statements := make([]tsgo.Statement, 0, len(source.List))
	var requests []api.RootRequest
	for _, statement := range source.List {
		target, err := children.Statement(context.WithRole(api.RoleBlockStatement), statement)
		if err != nil {
			return api.BlockEmission{}, err
		}
		statements = append(statements, target.Statements()...)
		requests = append(requests, target.Requests()...)
	}
	return api.DirectBlock(
		context.Factory().Block(statements, true),
		requests...,
	), nil
}
