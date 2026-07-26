package emit

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/statement/assignment"
	blockstatement "github.com/tsoniclang/gotots/internal/emit/statement/block"
	branchstatement "github.com/tsoniclang/gotots/internal/emit/statement/branch"
	expressionstatement "github.com/tsoniclang/gotots/internal/emit/statement/expressionstatement"
	forstatement "github.com/tsoniclang/gotots/internal/emit/statement/forstatement"
	ifstatement "github.com/tsoniclang/gotots/internal/emit/statement/ifstatement"
	incdecstatement "github.com/tsoniclang/gotots/internal/emit/statement/incdec"
	localdeclaration "github.com/tsoniclang/gotots/internal/emit/statement/localdeclaration"
	returnstatement "github.com/tsoniclang/gotots/internal/emit/statement/returnstatement"
	switchstatement "github.com/tsoniclang/gotots/internal/emit/statement/switchstatement"
)

func (e *emitter) Block(
	context api.Context,
	source *ast.BlockStmt,
) (api.BlockEmission, error) {
	return blockstatement.Emit(context, e, source)
}

func (e *emitter) Statement(
	context api.Context,
	source ast.Stmt,
) (api.StatementEmission, error) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		return assignment.Emit(context, e, source)
	case *ast.BlockStmt:
		target, err := blockstatement.Emit(context, e, source)
		if err != nil {
			return api.StatementEmission{}, err
		}
		return api.DirectStatement(target.Value(), target.Requests()...), nil
	case *ast.BranchStmt:
		return branchstatement.Emit(context, source)
	case *ast.DeclStmt:
		return localdeclaration.Emit(context, e, source)
	case *ast.ExprStmt:
		return expressionstatement.Emit(context, e, source)
	case *ast.ForStmt:
		return forstatement.Emit(context, e, source)
	case *ast.IfStmt:
		return ifstatement.Emit(context, e, source)
	case *ast.IncDecStmt:
		return incdecstatement.Emit(context, source)
	case *ast.ReturnStmt:
		return returnstatement.Emit(context, e, source)
	case *ast.SwitchStmt:
		return switchstatement.Emit(context, e, source)
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func (e *emitter) ScopedInitializer(
	context api.Context,
	source ast.Stmt,
) (api.StatementEmission, error) {
	initializer, ok := source.(*ast.AssignStmt)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return assignment.Emit(context, e, initializer)
}

func (e *emitter) IfAlternate(
	context api.Context,
	source *ast.IfStmt,
) (api.StatementEmission, error) {
	return ifstatement.Emit(context, e, source)
}

func (e *emitter) ForInitializer(
	context api.Context,
	source ast.Stmt,
) (api.ForInitializerEmission, error) {
	assignmentStatement, ok := source.(*ast.AssignStmt)
	if !ok {
		return api.ForInitializerEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return assignment.EmitForInitializer(context, e, assignmentStatement)
}

func (e *emitter) ForPost(
	context api.Context,
	source ast.Stmt,
) (api.ExpressionEmission, error) {
	post, ok := source.(*ast.IncDecStmt)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return incdecstatement.EmitExpression(context, post)
}
