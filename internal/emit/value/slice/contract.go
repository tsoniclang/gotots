package slicevalue

import (
	"go/types"
)

func Resolve(
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	if sourceType == nil {
		return nil, nil, false
	}
	sourceSlice, ok := types.Unalias(sourceType).(*types.Slice)
	if !ok || !directElement(sourceSlice.Elem()) {
		return nil, nil, false
	}
	return sourceSlice, sourceSlice.Elem(), true
}

func directElement(sourceType types.Type) bool {
	switch source := types.Unalias(sourceType).(type) {
	case *types.Basic, *types.Slice:
		return true
	case *types.Named:
		_, ok := source.Underlying().(*types.Basic)
		return ok
	default:
		return false
	}
}
