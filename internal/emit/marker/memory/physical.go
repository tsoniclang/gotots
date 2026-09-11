package memory

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	descriptorvalue "github.com/tsoniclang/gotots/internal/emit/value/memorydescriptor"
)

func SupportsLayout(context api.Context, source types.Type) (bool, error) {
	if source == nil || api.ContainsGenericTypeParameter(source) {
		return false, nil
	}
	if physicalLeaf(source) {
		return true, nil
	}
	if _, ok := descriptorvalue.Resolve(context, source); ok {
		return true, nil
	}
	if _, ok := complexvalue.Describe(source.Underlying()); ok {
		return true, nil
	}
	if array, ok := source.Underlying().(*types.Array); ok {
		return SupportsLayout(context, array.Elem())
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

func SupportsPointerTransport(context api.Context, source types.Type) (bool, error) {
	return supportsPointerTransport(context, source, make(map[types.Type]bool))
}

func supportsPointerTransport(context api.Context, source types.Type, active map[types.Type]bool) (bool, error) {
	supported, err := SupportsLayout(context, source)
	if err != nil || !supported {
		return false, err
	}
	if active[source] {
		return false, nil
	}
	active[source] = true
	defer delete(active, source)
	switch underlying := source.Underlying().(type) {
	case *types.Slice:
		return supportsPointerTransport(context, underlying.Elem(), active)
	case *types.Array:
		return supportsPointerTransport(context, underlying.Elem(), active)
	case *types.Struct:
		for index := range underlying.NumFields() {
			available, err := supportsPointerTransport(context, underlying.Field(index).Type(), active)
			if err != nil || !available {
				return false, err
			}
		}
	}
	return true, nil
}

func RequiresProjection(context api.Context, source types.Type) (bool, error) {
	if _, selected := descriptorvalue.Resolve(context, source); selected {
		return true, nil
	}
	switch source.Underlying().(type) {
	case *types.Array, *types.Struct:
		return true, nil
	}
	return context.Values().RequiresStorageProjection(context, source)
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
