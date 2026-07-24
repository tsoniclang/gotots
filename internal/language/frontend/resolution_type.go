package frontend

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (builder *packageBuilder) typeResolution(
	reference packageOccurrenceRef,
	record *occurrenceInput,
	context occurrenceContext,
) (identity.SemanticTypeID, bool, error) {
	expression, ok := record.node.(ast.Expr)
	if !ok {
		return identity.SemanticTypeID{}, false, nil
	}
	value, present := builder.input.loaded.CheckerView().TypeOf(expression)
	var resolved types.Type
	if present && value.IsType() {
		resolved = value.Type
	}
	if resolved == nil &&
		record.occurrence.Kind() == catalog.KindFuncType &&
		record.occurrence.Role() == catalog.RoleFunctionSignature {
		resolved = context.coverageType
		if resolved == nil {
			resolved = context.signature
		}
	}
	if resolved == nil {
		resolved = builder.inferredArrayEllipsisType(
			reference, record,
		)
	}
	if resolved == nil {
		return identity.SemanticTypeID{}, false, nil
	}
	typeID, err := builder.types.build(resolved)
	return typeID, true, err
}

func (builder *packageBuilder) inferredArrayEllipsisType(
	reference packageOccurrenceRef,
	record *occurrenceInput,
) types.Type {
	if record.occurrence.Kind() != catalog.KindEllipsis ||
		record.occurrence.Role() != catalog.RoleArrayLength {
		return nil
	}
	ellipsis, ellipsisNode := record.node.(*ast.Ellipsis)
	parent := builder.input.occurrenceRecord(
		builder.input.occurrenceParent(reference),
	)
	if !ellipsisNode || parent == nil {
		return nil
	}
	array, arrayNode := parent.node.(*ast.ArrayType)
	if !arrayNode || array.Len != ellipsis {
		return nil
	}
	value, present := builder.input.loaded.CheckerView().TypeOf(array)
	if !present || !value.IsType() {
		return nil
	}
	underlying, inferred := types.Unalias(
		value.Type,
	).Underlying().(*types.Array)
	if !inferred || underlying.Len() < 0 {
		return nil
	}
	return value.Type
}
