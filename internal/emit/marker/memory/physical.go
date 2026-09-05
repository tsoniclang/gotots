package memory

import "go/types"

func physicalLayoutRepresentable(source types.Type) bool {
	if physicalLeaf(source) {
		return true
	}
	structure, ok := source.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for index := range structure.NumFields() {
		field := structure.Field(index)
		if field.Name() == "_" || !physicalLeaf(field.Type()) {
			return false
		}
	}
	return true
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
