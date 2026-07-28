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
	if !ok {
		return nil, nil, false
	}
	return sourceSlice, sourceSlice.Elem(), true
}
