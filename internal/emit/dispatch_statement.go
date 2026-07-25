package emit

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	blockstatement "github.com/tsoniclang/gotots/internal/emit/statement/block"
	returnstatement "github.com/tsoniclang/gotots/internal/emit/statement/returnstatement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *Emitter) Block(context api.Context, source *ast.BlockStmt) (tsgo.Block, error) {
	return blockstatement.Emit(context, e, source)
}

func (e *Emitter) Statement(context api.Context, source ast.Stmt) (tsgo.Statement, error) {
	switch source := source.(type) {
	case *ast.ReturnStmt:
		return returnstatement.Emit(context, e, source)
	default:
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
}
