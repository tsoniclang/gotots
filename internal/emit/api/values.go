package api

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Values interface {
	RequiresCustomEquality(Context, types.Type) bool
	RequiresCustomUpdate(Context, types.Type) bool
	RequiresExplicitType(Context, types.Type) bool
	RequiresStructuralCopy(Context, types.Type) bool
	Zero(Context, ast.Node, types.Type) (ExpressionEmission, error)
	Copy(Context, ast.Node, types.Type, ExpressionEmission) (ExpressionEmission, error)
	Assign(
		Context,
		ast.Node,
		types.Type,
		tsgo.Expression,
		ExpressionEmission,
	) (ExpressionEmission, error)
	Equal(
		Context,
		ast.Node,
		types.Type,
		tsgo.Expression,
		tsgo.Expression,
	) (ExpressionEmission, error)
	BinaryUpdate(
		Context,
		ast.Node,
		ast.Expr,
		types.Type,
		types.Type,
		token.Token,
		tsgo.Expression,
		ExpressionEmission,
	) (ExpressionEmission, bool, error)
	Increment(
		Context,
		ast.Node,
		types.Type,
		token.Token,
		tsgo.Expression,
	) (ExpressionEmission, bool, error)
}
