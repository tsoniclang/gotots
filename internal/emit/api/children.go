package api

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type ChildEmitter interface {
	Block(Context, *ast.BlockStmt) (tsgo.Block, error)
	Statement(Context, ast.Stmt) (tsgo.Statement, error)
	Expression(Context, ast.Expr) (tsgo.Expression, error)
	Condition(Context, ast.Expr) (tsgo.Expression, error)
	Type(Context, ast.Expr) (tsgo.TypeNode, error)
}
