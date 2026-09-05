package rawpointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
)

func Convert(
	context api.Context,
	children api.ChildEmitter,
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
		if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
			layout, pointee, err := memorymarker.Layout(context, children, source, pointer.Elem())
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
			target, err := pointermarker.Operation(context, tsoniccore.SymbolToRawPointer, []api.TypeEmission{pointee}, []api.ExpressionEmission{value, layout})
			return target, true, err
		}
	}
	if sourceRaw {
		if pointer, ok := types.Unalias(targetType).(*types.Pointer); ok {
			layout, pointee, err := memorymarker.Layout(context, children, source, pointer.Elem())
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
			target, err := pointermarker.Operation(context, tsoniccore.SymbolReinterpretRawPointer, []api.TypeEmission{pointee}, []api.ExpressionEmission{value, layout})
			return target, true, err
		}
	}
	return api.ExpressionEmission{}, true,
		api.Unsupported(context, api.CategoryExpression, source)
}
