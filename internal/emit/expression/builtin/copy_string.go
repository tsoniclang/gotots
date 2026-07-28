package builtin

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitCopyFromString(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.CallExpr,
	_ bool,
) (api.ExpressionEmission, error) {
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryExpression, source)
}
