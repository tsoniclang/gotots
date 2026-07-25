package emit

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	blockstatement "github.com/tsoniclang/gotots/internal/emit/statement/block"
	branchstatement "github.com/tsoniclang/gotots/internal/emit/statement/branch"
	forstatement "github.com/tsoniclang/gotots/internal/emit/statement/forstatement"
	ifstatement "github.com/tsoniclang/gotots/internal/emit/statement/ifstatement"
	incdecstatement "github.com/tsoniclang/gotots/internal/emit/statement/incdec"
	returnstatement "github.com/tsoniclang/gotots/internal/emit/statement/returnstatement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *Emitter) Block(context api.Context, source *ast.BlockStmt) (tsgo.Block, error) {
	return blockstatement.Emit(context, e, source)
}

func (e *Emitter) Statement(context api.Context, source ast.Stmt) (tsgo.Statement, error) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		return assignment.Emit(context, e, source)
	case *ast.BranchStmt:
		return branchstatement.Emit(context, source)
	case *ast.ForStmt:
		return forstatement.Emit(context, e, source)
	case *ast.IfStmt:
		return ifstatement.Emit(context, e, source)
	case *ast.IncDecStmt:
		return incdecstatement.Emit(context, source)
	case *ast.ReturnStmt:
		return returnstatement.Emit(context, e, source)
	default:
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
}

func (e *Emitter) ForInitializer(
	context api.Context,
	source ast.Stmt,
) (tsgo.ForInitializer, error) {
	assignmentStatement, ok := source.(*ast.AssignStmt)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	return assignment.EmitForInitializer(context, e, assignmentStatement)
}

func (e *Emitter) ForPost(
	context api.Context,
	source ast.Stmt,
) (tsgo.Expression, error) {
	post, ok := source.(*ast.IncDecStmt)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	return incdecstatement.EmitExpression(context, post)
}
