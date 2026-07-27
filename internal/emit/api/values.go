package api

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Values interface {
	RequiresCustomEquality(Context, types.Type) bool
	RequiresExplicitType(Context, types.Type) bool
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
}
