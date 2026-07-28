package stringconversion

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Convert(
	context api.Context,
	source *ast.CallExpr,
	_ types.Type,
	_ types.Type,
	_ api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryExpression, source)
}
