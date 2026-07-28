package slicevalue

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
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

func Source(
	sourceType types.Type,
) (*types.Slice, types.Type, bool) {
	if defined, ok := definedtype.ResolveSlice(sourceType); ok {
		sliceType, _ := defined.Slice()
		return sliceType, sliceType.Elem(), true
	}
	return Resolve(sourceType)
}

func Project(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	defined, ok := definedtype.ResolveSlice(sourceType)
	if !ok {
		return value, nil
	}
	return defined.Project(context, value)
}

func Wrap(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	defined, ok := definedtype.ResolveSlice(sourceType)
	if !ok {
		return value, nil
	}
	return defined.Wrap(context, value)
}
