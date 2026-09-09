package memory

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func SupportsLayout(context api.Context, source types.Type) (bool, error) {
	if source == nil || api.ContainsGenericTypeParameter(source) {
		return false, nil
	}
	if physicalLeaf(source) {
		return true, nil
	}
	structure, ok := source.Underlying().(*types.Struct)
	if !ok {
		return false, nil
	}
	projected, err := context.Values().RequiresStorageProjection(context, source)
	if err != nil || !projected {
		return false, err
	}
	for index := range structure.NumFields() {
		field := structure.Field(index)
		supported, err := SupportsLayout(context, field.Type())
		if err != nil || !supported {
			return false, err
		}
	}
	return true, nil
}

func physicalLeaf(source types.Type) bool {
	switch underlying := source.Underlying().(type) {
	case *types.Pointer:
		return true
	case *types.Basic:
		return underlying.Info()&types.IsUntyped == 0 &&
			(underlying.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat) != 0 || underlying.Kind() == types.UnsafePointer)
	default:
		return false
	}
}
