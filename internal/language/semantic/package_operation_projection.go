package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func storedRange[Value any](
	values []Value,
	start uint64,
	count uint64,
) ([]Value, error) {
	if start > uint64(len(values)) ||
		count > uint64(len(values))-start {
		return nil, fmt.Errorf(
			"semantic package range %d+%d exceeds %d",
			start, count, len(values),
		)
	}
	return values[start : start+count], nil
}

func (store packageOperationStore) object(
	identities *packageIdentityProjection,
	reference objectReferenceRef,
) (ObjectReference, error) {
	if reference == 0 {
		return NoObjectReference(), nil
	}
	stored, err := payloadAt(store.objects, uint64(reference))
	if err != nil {
		return ObjectReference{}, err
	}
	return ObjectReference{
		kind: stored.kind,
		declaration: identities.declaration(
			stored.declaration,
		),
		binding: identities.binding(stored.binding),
	}, nil
}

func (store packageOperationStore) constant(
	reference constantRef,
) (Constant, error) {
	if reference == 0 {
		return Constant{}, nil
	}
	stored, err := payloadAt(store.constants, uint64(reference))
	if err != nil {
		return Constant{}, err
	}
	return Constant{kind: stored.kind, exact: stored.exact}, nil
}

func (store packageOperationStore) selection(
	identities *packageIdentityProjection,
	reference selectionRef,
) (Selection, error) {
	if reference == 0 {
		return Selection{}, nil
	}
	stored, err := payloadAt(store.selections, uint64(reference))
	if err != nil {
		return Selection{}, err
	}
	index, err := storedRange(
		store.selectionIndexes,
		stored.index.start,
		stored.index.count,
	)
	if err != nil {
		return Selection{}, err
	}
	return Selection{
		kind:     stored.kind,
		receiver: identities.typeID(stored.receiver),
		object:   identities.declaration(stored.object),
		index:    append([]int(nil), index...),
		indirect: stored.indirect,
	}, nil
}

func (store packageOperationStore) instance(
	identities *packageIdentityProjection,
	reference instanceRef,
) (Instance, error) {
	if reference == 0 {
		return Instance{}, nil
	}
	stored, err := payloadAt(store.instances, uint64(reference))
	if err != nil {
		return Instance{}, err
	}
	target, err := store.object(identities, stored.target)
	if err != nil {
		return Instance{}, err
	}
	references, err := storedRange(
		store.instanceTypes,
		stored.types.start,
		stored.types.count,
	)
	if err != nil {
		return Instance{}, err
	}
	types := make([]identity.SemanticTypeID, len(references))
	for index, reference := range references {
		types[index] = identities.typeID(reference)
	}
	return Instance{
		target: target, types: types,
		signature: identities.typeID(stored.signature),
	}, nil
}

type packageOperationProjection struct {
	store      packageOperationStore
	identities packageIdentityTable
}

func newPackageOperationProjection(
	store packageOperationStore,
	identities packageIdentityTable,
) *packageOperationProjection {
	return &packageOperationProjection{
		store: store, identities: identities,
	}
}

func (projection *packageOperationProjection) operation(
	identities *packageIdentityProjection,
	index int,
) (Operation, error) {
	if projection == nil ||
		identities == nil ||
		index < 0 ||
		index >= len(projection.store.records) {
		return Operation{}, fmt.Errorf(
			"semantic operation index %d is invalid", index,
		)
	}
	stored := projection.store.records[index]
	constant, err := projection.store.constant(stored.constant)
	if err != nil {
		return Operation{}, err
	}
	object, err := projection.store.object(identities, stored.object)
	if err != nil {
		return Operation{}, err
	}
	return Operation{
		spec: OperationSpec{
			ID:            identities.operation(stored.id),
			Kind:          stored.kind,
			Syntax:        stored.syntax,
			Variant:       stored.variant,
			Role:          stored.role,
			Token:         stored.token,
			Mode:          stored.mode,
			Arity:         stored.arity,
			Place:         stored.place,
			ResultType:    identities.typeID(stored.resultType),
			ExpectedType:  identities.typeID(stored.expectedType),
			Addressable:   stored.addressable,
			Assignable:    stored.assignable,
			HasOk:         stored.hasOk,
			Constant:      constant,
			Object:        object,
			ControlTarget: identities.operation(stored.controlTarget),
			Label:         identities.binding(stored.label),
		},
		projection: projection,
		index:      index,
	}, nil
}
