package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitConstantMeasure(
	context api.Context,
	_ *ast.CallExpr,
	_ *types.Builtin,
	_ bool,
) (api.ExpressionEmission, bool, error) {
	return api.ExpressionEmission{}, false, nil
}
