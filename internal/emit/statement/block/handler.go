package block

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BlockStmt,
) (api.BlockEmission, error) {
	if source == nil {
		return api.BlockEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	target, err := children.Statements(
		context.WithRole(api.RoleBlockStatement),
		source,
		source.List,
	)
	if err != nil {
		return api.BlockEmission{}, err
	}
	return api.DirectBlock(
		context.Factory().Block(target.Statements(), true),
		target.Requests()...,
	), nil
}
