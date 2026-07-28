package newvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitExpression(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.CallExpr,
	_ *types.Builtin,
) (api.ExpressionEmission, error) {
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryExpression, source)
}
