package api

import (
	"go/ast"
	"go/types"
)

type ChildEmitter interface {
	Block(Context, *ast.BlockStmt) (BlockEmission, error)
	Statement(Context, ast.Stmt) (StatementEmission, error)
	Expression(Context, ast.Expr) (ExpressionEmission, error)
	DiscardedCall(Context, *ast.CallExpr) (ExpressionEmission, error)
	Condition(Context, ast.Expr) (ExpressionEmission, error)
	IntegerConstant(Context, ast.Expr) (ExpressionEmission, error)
	ScopedInitializer(Context, ast.Stmt) (StatementEmission, error)
	IfAlternate(Context, *ast.IfStmt) (StatementEmission, error)
	ForInitializer(Context, ast.Stmt) (ForInitializerEmission, error)
	ForPost(Context, ast.Stmt) (ExpressionEmission, error)
	Type(Context, ast.Expr) (TypeEmission, error)
	RepresentedType(Context, ast.Node, types.Type) (TypeEmission, error)
}
