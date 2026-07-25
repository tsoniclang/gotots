package api

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type ChildEmitter interface {
	Block(Context, *ast.BlockStmt) (tsgo.Block, error)
	Statement(Context, ast.Stmt) (tsgo.Statement, error)
	Expression(Context, ast.Expr) (tsgo.Expression, error)
	Condition(Context, ast.Expr) (tsgo.Expression, error)
	IntegerConstant(Context, ast.Expr) (tsgo.Expression, error)
	ForInitializer(Context, ast.Stmt) (tsgo.ForInitializer, error)
	ForPost(Context, ast.Stmt) (tsgo.Expression, error)
	Type(Context, ast.Expr) (tsgo.TypeNode, error)
	RepresentedType(Context, ast.Node, types.Type) (tsgo.TypeNode, error)
}
