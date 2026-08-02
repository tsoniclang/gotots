package api

import (
	"go/ast"
	"go/types"
)

type ContextualStructField struct {
	declaration *types.Var
	selected    *types.Var
	index       int
}

func (f ContextualStructField) Declaration() *types.Var {
	return f.declaration
}

func (f ContextualStructField) Selected() *types.Var {
	return f.selected
}

func (f ContextualStructField) Index() int {
	return f.index
}

func (v TypeInfoView) StructFieldAt(
	source *ast.CompositeLit,
	index int,
) (ContextualStructField, bool) {
	declaration, selected, ok := v.structFields(source)
	if !ok || index < 0 || index >= declaration.NumFields() {
		return ContextualStructField{}, false
	}
	return contextualStructField(declaration, selected, index)
}

func (v TypeInfoView) StructFieldOf(
	source *ast.CompositeLit,
	key *ast.Ident,
) (ContextualStructField, bool) {
	declaration, selected, ok := v.structFields(source)
	field, fieldOK := v.UseOf(key).(*types.Var)
	if !ok || !fieldOK {
		return ContextualStructField{}, false
	}
	for index := range declaration.NumFields() {
		if declaration.Field(index) == field || selected.Field(index) == field {
			return contextualStructField(declaration, selected, index)
		}
	}
	return ContextualStructField{}, false
}

func (v TypeInfoView) structFields(
	source *ast.CompositeLit,
) (*types.Struct, *types.Struct, bool) {
	if v.source == nil || source == nil {
		return nil, nil, false
	}
	declaration, selected, ok := contextualStructTypes(v.TypeOf(source))
	return declaration, selected,
		ok && declaration.NumFields() == selected.NumFields()
}

func contextualStructField(
	declaration *types.Struct,
	selected *types.Struct,
	index int,
) (ContextualStructField, bool) {
	sourceField := declaration.Field(index)
	targetField := selected.Field(index)
	if sourceField.Id() != targetField.Id() ||
		sourceField.Embedded() != targetField.Embedded() ||
		declaration.Tag(index) != selected.Tag(index) {
		return ContextualStructField{}, false
	}
	return ContextualStructField{
		declaration: sourceField,
		selected:    targetField,
		index:       index,
	}, true
}

func contextualStructTypes(
	source types.Type,
) (*types.Struct, *types.Struct, bool) {
	if source == nil {
		return nil, nil, false
	}
	selectedType := types.Unalias(source)
	if pointer, ok := selectedType.(*types.Pointer); ok {
		selectedType = types.Unalias(pointer.Elem())
	}
	if named, ok := selectedType.(*types.Named); ok {
		selected, selectedOK := named.Underlying().(*types.Struct)
		declaration, declarationOK := named.Origin().Underlying().(*types.Struct)
		return declaration, selected, declarationOK && selectedOK
	}
	selected, ok := selectedType.Underlying().(*types.Struct)
	return selected, selected, ok
}
