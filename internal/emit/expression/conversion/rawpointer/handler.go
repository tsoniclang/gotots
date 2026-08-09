package rawpointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	rawpointermarker "github.com/tsoniclang/gotots/internal/emit/marker/rawpointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
)

func Convert(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	targetRaw := basictype.SupportsUnsafePointer(targetType)
	sourceRaw := basictype.SupportsUnsafePointer(sourceType)
	if !targetRaw && !sourceRaw {
		return api.ExpressionEmission{}, false, nil
	}
	if targetRaw && sourceRaw {
		return value, true, nil
	}
	if targetRaw {
		if _, ok := types.Unalias(sourceType).(*types.Pointer); ok {
			target, err := rawpointermarker.BindNullable(context, value)
			return target, true, err
		}
	}
	return api.ExpressionEmission{}, true,
		api.Unsupported(context, api.CategoryExpression, source)
}
