package clear

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	_ bool,
) (api.ExpressionEmission, bool, error) {
	if builtin == nil ||
		types.Object(builtin) != types.Universe.Lookup("clear") {
		return api.ExpressionEmission{}, false, nil
	}
	return api.ExpressionEmission{}, true,
		api.Unsupported(context, api.CategoryExpression, source)
}
