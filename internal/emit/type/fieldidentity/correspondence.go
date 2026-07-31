package fieldidentity

import "go/types"

type Correspondence struct {
	owner       *types.TypeName
	declaration types.Type
	selected    types.Type
}

type ResolutionError struct {
	Reason string
}

func (e *ResolutionError) Error() string {
	return "generic named-struct field correspondence: " + e.Reason
}

func Resolve(
	container types.Type,
	field *types.Var,
) (Correspondence, bool, error) {
	container = indirect(types.Unalias(container))
	named, ok := container.(*types.Named)
	if !ok || named == nil || named.Obj() == nil {
		return Correspondence{}, false, nil
	}
	origin := named.Origin()
	if origin == nil ||
		origin.Obj() == nil ||
		origin.TypeParams().Len() == 0 {
		return Correspondence{}, false, nil
	}
	if named.TypeArgs().Len() != named.TypeParams().Len() {
		return Correspondence{}, false, &ResolutionError{
			Reason: "receiver is not fully instantiated",
		}
	}
	selectedStruct, selectedOK := named.Underlying().(*types.Struct)
	originStruct, originOK := origin.Underlying().(*types.Struct)
	if !selectedOK || !originOK ||
		selectedStruct.NumFields() != originStruct.NumFields() {
		return Correspondence{}, false, &ResolutionError{
			Reason: "receiver does not preserve its origin struct",
		}
	}
	index := fieldIndex(selectedStruct, field)
	if index < 0 {
		return Correspondence{}, false, &ResolutionError{
			Reason: "selected field is not owned by the receiver",
		}
	}
	declarationField := originStruct.Field(index)
	selectedField := selectedStruct.Field(index)
	if declarationField.Id() != selectedField.Id() ||
		declarationField.Embedded() != selectedField.Embedded() ||
		originStruct.Tag(index) != selectedStruct.Tag(index) {
		return Correspondence{}, false, &ResolutionError{
			Reason: "field ordinal no longer preserves its declaration identity",
		}
	}
	return Correspondence{
		owner:       origin.Obj(),
		declaration: declarationField.Type(),
		selected:    selectedField.Type(),
	}, true, nil
}

func (c Correspondence) Parts() (
	*types.TypeName,
	types.Type,
	types.Type,
) {
	return c.owner, c.declaration, c.selected
}

func fieldIndex(structType *types.Struct, field *types.Var) int {
	if structType == nil || field == nil {
		return -1
	}
	for index := range structType.NumFields() {
		if structType.Field(index) == field {
			return index
		}
	}
	return -1
}

func indirect(source types.Type) types.Type {
	if pointer, ok := source.(*types.Pointer); ok {
		return types.Unalias(pointer.Elem())
	}
	return source
}
