package rawpointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
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
	if types.Identical(sourceType, targetType) {
		return value, true, nil
	}
	var err error
	if defined, ok := definedtype.Resolve(sourceType); ok {
		value, err = defined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		sourceType = defined.Underlying()
	}
	targetDefined, wrapsTarget := definedtype.Resolve(targetType)
	if wrapsTarget {
		targetType = targetDefined.Underlying()
	}
	target := value
	if !sourceRaw || !targetRaw {
		pointerType := sourceType
		operation := tsoniccore.SymbolToRawPointer
		if sourceRaw {
			pointerType = targetType
			operation = tsoniccore.SymbolReinterpretRawPointer
		}
		pointer, ok := types.Unalias(pointerType).(*types.Pointer)
		if !ok {
			return api.ExpressionEmission{}, true, api.Unsupported(context, api.CategoryExpression, source)
		}
		layout, pointee, layoutErr := memorymarker.Layout(context, children, source, pointer.Elem())
		if layoutErr != nil {
			return api.ExpressionEmission{}, true, layoutErr
		}
		target, err = pointermarker.Operation(context, operation, []api.TypeEmission{pointee}, []api.ExpressionEmission{value, layout})
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	if wrapsTarget {
		target, err = targetDefined.Wrap(context, target)
	}
	return target, true, err
}
