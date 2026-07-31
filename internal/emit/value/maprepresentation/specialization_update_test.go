package maprepresentation

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (v staticSpecializationValues) BinaryUpdate(
	api.Context,
	ast.Node,
	ast.Expr,
	types.Type,
	types.Type,
	token.Token,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}

func (v staticSpecializationValues) Increment(
	api.Context,
	ast.Node,
	types.Type,
	token.Token,
	tsgo.Expression,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}
