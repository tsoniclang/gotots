package stringvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

type sliceKind uint8

const (
	sliceInvalid sliceKind = iota
	sliceBytes
	sliceRunes
)

func Convert(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	sourceString := basictype.SupportsString(sourceType)
	targetString := basictype.SupportsString(targetType)
	sourceSlice, sourceElement, sourceSliceKind := classifySlice(sourceType)
	targetSlice, targetElement, targetSliceKind := classifySlice(targetType)
	switch {
	case targetString && integerSource(context, sourceType):
		target, err := integerToString(context, operand)
		return target, true, err
	case targetString && sourceSlice != nil:
		target, err := sliceToString(
			context,
			source,
			sourceSlice,
			sourceElement,
			sourceSliceKind,
			operand,
		)
		return target, true, err
	case sourceString && targetSlice != nil:
		target, err := stringToSlice(
			context,
			children,
			source,
			targetSlice,
			targetElement,
			targetSliceKind,
			operand,
		)
		return target, true, err
	default:
		return api.ExpressionEmission{}, false, nil
	}
}

func integerSource(context api.Context, sourceType types.Type) bool {
	_, ok := integervalue.Describe(context.TypesSizes(), sourceType)
	return ok
}

func classifySlice(
	sourceType types.Type,
) (*types.Slice, types.Type, sliceKind) {
	source, ok := types.Unalias(sourceType).(*types.Slice)
	if !ok {
		return nil, nil, sliceInvalid
	}
	element := source.Elem()
	basic, ok := types.Unalias(element).(*types.Basic)
	if !ok {
		return nil, nil, sliceInvalid
	}
	switch basic.Kind() {
	case types.Uint8:
		return source, element, sliceBytes
	case types.Int32:
		return source, element, sliceRunes
	default:
		return nil, nil, sliceInvalid
	}
}
