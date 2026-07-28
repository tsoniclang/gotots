package slicearray

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
)

func Convert(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	sourceSlice, sourceElement, sourceOK := slicevalue.Resolve(sourceType)
	targetArray, targetOK := arrayvalue.Resolve(context, targetType)
	if !sourceOK || !targetOK {
		return api.ExpressionEmission{}, false, nil
	}
	if sourceSlice == nil ||
		!types.Identical(sourceElement, targetArray.ElementType()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := targetArray.FromSlice(
		context,
		children,
		source,
		operand,
	)
	return target, true, err
}
