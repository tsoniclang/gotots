package slicevalue

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Scalar(
	sizes types.Sizes,
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	if sizes == nil || sourceType == nil {
		return nil, nil, false
	}
	sourceSlice, ok := types.Unalias(sourceType).(*types.Slice)
	if !ok {
		return nil, nil, false
	}
	elementType := sourceSlice.Elem()
	if _, represented := api.PrimitiveAliasFor(sizes, elementType); !represented {
		return nil, nil, false
	}
	return sourceSlice, elementType, true
}
