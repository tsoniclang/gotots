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
) (tsgo.Block, error) {
	statements := make([]tsgo.Statement, 0, len(source.List))
	for _, statement := range source.List {
		target, err := children.Statement(context.WithRole(api.RoleBlockStatement), statement)
		if err != nil {
			return nil, err
		}
		statements = append(statements, target)
	}
	return context.Factory().Block(statements, true), nil
}
