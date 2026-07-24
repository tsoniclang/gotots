package semantic

import (
	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

func (operation Operation) stored() (storedOperation, bool) {
	if operation.projection == nil ||
		operation.index < 0 ||
		operation.index >= len(operation.projection.store.records) {
		return storedOperation{}, false
	}
	return operation.projection.store.records[operation.index], true
}

func (operation Operation) ID() identity.OperationID {
	return operation.spec.ID
}

func (operation Operation) Definition() identity.DefinitionID {
	return operation.ID().Definition()
}

func (operation Operation) Occurrence() identity.OccurrenceID {
	return operation.ID().Occurrence()
}

func (operation Operation) Kind() OperationKind {
	return operation.spec.Kind
}

func (operation Operation) Syntax() catalog.Kind {
	return operation.spec.Syntax
}

func (operation Operation) Variant() catalog.Variant {
	return operation.spec.Variant
}

func (operation Operation) Role() catalog.Role {
	return operation.spec.Role
}

func (operation Operation) Token() catalog.TokenKind {
	return operation.spec.Token
}

func (operation Operation) Mode() ValueMode {
	return operation.spec.Mode
}

func (operation Operation) Arity() ResultArity {
	return operation.spec.Arity
}

func (operation Operation) Place() PlaceKind {
	return operation.spec.Place
}

func (operation Operation) ResultType() identity.SemanticTypeID {
	return operation.spec.ResultType
}

func (operation Operation) ExpectedType() identity.SemanticTypeID {
	return operation.spec.ExpectedType
}

func (operation Operation) Addressable() bool {
	return operation.spec.Addressable
}

func (operation Operation) Assignable() bool {
	return operation.spec.Assignable
}

func (operation Operation) HasOk() bool {
	return operation.spec.HasOk
}

func (operation Operation) Constant() Constant {
	return operation.spec.Constant
}

func (operation Operation) Object() ObjectReference {
	return operation.spec.Object
}

func (operation Operation) Selection() Selection {
	if stored, present := operation.stored(); present {
		identities := newPackageIdentityProjection(
			operation.projection.identities,
		)
		value, _ := operation.projection.store.selection(
			identities,
			stored.selection,
		)
		return value
	}
	return operation.spec.Selection
}

func (operation Operation) Instance() Instance {
	if stored, present := operation.stored(); present {
		identities := newPackageIdentityProjection(
			operation.projection.identities,
		)
		value, _ := operation.projection.store.instance(
			identities,
			stored.instance,
		)
		return value
	}
	return operation.spec.Instance
}

func (operation Operation) ControlTarget() identity.OperationID {
	return operation.spec.ControlTarget
}

func (operation Operation) Label() identity.SemanticBindingID {
	return operation.spec.Label
}

func (operation Operation) OperandCount() int {
	if stored, present := operation.stored(); present {
		return int(stored.operands.count)
	}
	return len(operation.spec.Operands)
}

func (operation Operation) Operand(
	index int,
) (identity.OccurrenceID, bool) {
	if stored, present := operation.stored(); present {
		reference, ok := operationReferenceAt(
			operation.projection.store.operands,
			stored.operands.start,
			stored.operands.count,
			index,
		)
		if !ok {
			return identity.OccurrenceID{}, false
		}
		return operation.projection.identities.occurrence(
			reference,
		), true
	}
	if index < 0 ||
		index >= len(operation.spec.Operands) {
		return identity.OccurrenceID{}, false
	}
	return operation.spec.Operands[index], true
}

func (operation Operation) NestedDefinitionCount() int {
	if stored, present := operation.stored(); present {
		return int(stored.definitions.count)
	}
	return len(operation.spec.Definitions)
}

func (operation Operation) NestedDefinition(
	index int,
) (identity.DefinitionID, bool) {
	if stored, present := operation.stored(); present {
		reference, ok := operationReferenceAt(
			operation.projection.store.definitions,
			stored.definitions.start,
			stored.definitions.count,
			index,
		)
		if !ok {
			return identity.DefinitionID{}, false
		}
		return operation.projection.identities.definition(
			reference,
		), true
	}
	if index < 0 ||
		index >= len(operation.spec.Definitions) {
		return identity.DefinitionID{}, false
	}
	return operation.spec.Definitions[index], true
}

func (operation Operation) ImplicitCount() int {
	if stored, present := operation.stored(); present {
		return int(stored.implicit.count)
	}
	return len(operation.spec.Implicit)
}

func (operation Operation) Implicit(
	index int,
) (ImplicitOperation, bool) {
	if stored, present := operation.stored(); present {
		record, ok := operationReferenceAt(
			operation.projection.store.implicit,
			stored.implicit.start,
			stored.implicit.count,
			index,
		)
		if !ok {
			return ImplicitOperation{}, false
		}
		return ImplicitOperation{
			kind: record.kind,
			site: operation.projection.identities.occurrence(
				record.site,
			),
			ordinal: record.ordinal,
			source: operation.projection.identities.typeID(
				record.source,
			),
			target: operation.projection.identities.typeID(
				record.target,
			),
		}, true
	}
	if index < 0 ||
		index >= len(operation.spec.Implicit) {
		return ImplicitOperation{}, false
	}
	return operation.spec.Implicit[index], true
}

func operationReferenceAt[Value any](
	values []Value,
	start uint64,
	count uint64,
	index int,
) (Value, bool) {
	if index < 0 ||
		uint64(index) >= count ||
		start > uint64(len(values)) ||
		count > uint64(len(values))-start {
		var zero Value
		return zero, false
	}
	return values[start+uint64(index)], true
}

func (operation Operation) Spec() OperationSpec {
	if operation.projection == nil {
		return cloneOperationSpec(operation.spec)
	}
	return operation.projectSpec()
}

func (operation Operation) normalizationSpec() OperationSpec {
	if operation.projection == nil {
		return operation.spec
	}
	return operation.projectSpec()
}

func (operation Operation) projectSpec() OperationSpec {
	spec := OperationSpec{
		ID:            operation.ID(),
		Kind:          operation.Kind(),
		Syntax:        operation.Syntax(),
		Variant:       operation.Variant(),
		Role:          operation.Role(),
		Token:         operation.Token(),
		Mode:          operation.Mode(),
		Arity:         operation.Arity(),
		Place:         operation.Place(),
		ResultType:    operation.ResultType(),
		ExpectedType:  operation.ExpectedType(),
		Addressable:   operation.Addressable(),
		Assignable:    operation.Assignable(),
		HasOk:         operation.HasOk(),
		Constant:      operation.Constant(),
		Object:        operation.Object(),
		Selection:     operation.Selection(),
		Instance:      operation.Instance(),
		ControlTarget: operation.ControlTarget(),
		Label:         operation.Label(),
	}
	spec.Operands = make(
		[]identity.OccurrenceID,
		operation.OperandCount(),
	)
	for index := range spec.Operands {
		spec.Operands[index], _ = operation.Operand(index)
	}
	spec.Definitions = make(
		[]identity.DefinitionID,
		operation.NestedDefinitionCount(),
	)
	for index := range spec.Definitions {
		spec.Definitions[index], _ = operation.NestedDefinition(index)
	}
	spec.Implicit = make(
		[]ImplicitOperation,
		operation.ImplicitCount(),
	)
	for index := range spec.Implicit {
		spec.Implicit[index], _ = operation.Implicit(index)
	}
	return spec
}
