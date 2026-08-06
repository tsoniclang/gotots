package abi

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type slotKind uint8

const (
	slotInvalid slotKind = iota
	slotCapability
	slotSourceParameter
)

type slot struct {
	kind      slotKind
	operation *api.GenericOperationContract
	parameter *types.Var
}

type Binding[T any] struct {
	slot  slot
	value T
}

func Capability[T any](
	operation *api.GenericOperationContract,
	value T,
) (Binding[T], error) {
	if !operation.Valid() {
		return Binding[T]{}, invariant(
			"generic capability ABI binding is invalid",
		)
	}
	return Binding[T]{
		slot: slot{
			kind:      slotCapability,
			operation: operation,
		},
		value: value,
	}, nil
}

func SourceParameters[T any](
	owner *types.Func,
	values []T,
) ([]Binding[T], error) {
	_, signature, err := method(owner)
	if err != nil {
		return nil, err
	}
	if signature.Params().Len() != len(values) {
		return nil, invariant(
			"generic method source ABI values do not match parameters",
		)
	}
	result := make([]Binding[T], 0, len(values))
	for index, value := range values {
		result = append(result, Binding[T]{
			slot: slot{
				kind:      slotSourceParameter,
				parameter: signature.Params().At(index),
			},
			value: value,
		})
	}
	return result, nil
}

func JoinCapabilities[T any](
	owner types.Object,
	operations []*api.GenericOperationContract,
	bindings []Binding[T],
) ([]T, error) {
	owner = api.GenericDeclarationOrigin(owner)
	if owner == nil {
		return nil, invariant("generic capability ABI owner is invalid")
	}
	expected, err := capabilitySlots(owner, operations)
	if err != nil {
		return nil, err
	}
	return join(expected, bindings)
}

func JoinClassMethod[T any](
	owner *types.Func,
	operations []*api.GenericOperationContract,
	bindings []Binding[T],
) ([]T, error) {
	owner, signature, err := method(owner)
	if err != nil {
		return nil, err
	}
	expected, err := capabilitySlots(owner, operations)
	if err != nil {
		return nil, err
	}
	for index := range signature.Params().Len() {
		expected = append(expected, slot{
			kind:      slotSourceParameter,
			parameter: signature.Params().At(index),
		})
	}
	return join(expected, bindings)
}

func Combine[T any](groups ...[]Binding[T]) []Binding[T] {
	var size int
	for _, group := range groups {
		size += len(group)
	}
	result := make([]Binding[T], 0, size)
	for _, group := range groups {
		result = append(result, group...)
	}
	return result
}

func capabilitySlots(
	owner types.Object,
	operations []*api.GenericOperationContract,
) ([]slot, error) {
	result := make([]slot, 0, len(operations))
	seen := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if !operation.Valid() ||
			operation.Owner() != owner {
			return nil, invariant(
				"generic capability ABI operation is invalid",
			)
		}
		if _, duplicate := seen[operation.Key()]; duplicate {
			return nil, invariant(
				"generic capability ABI operation is duplicated",
			)
		}
		seen[operation.Key()] = struct{}{}
		result = append(result, slot{
			kind:      slotCapability,
			operation: operation,
		})
	}
	return result, nil
}

func join[T any](
	expected []slot,
	bindings []Binding[T],
) ([]T, error) {
	if len(expected) != len(bindings) {
		return nil, invariant("generic ABI binding cardinality differs")
	}
	remaining := make(map[slot]T, len(bindings))
	for _, binding := range bindings {
		if binding.slot.kind == slotInvalid {
			return nil, invariant("generic ABI binding identity is invalid")
		}
		if _, duplicate := remaining[binding.slot]; duplicate {
			return nil, invariant("generic ABI binding identity is duplicated")
		}
		remaining[binding.slot] = binding.value
	}
	result := make([]T, 0, len(expected))
	for _, identity := range expected {
		value, present := remaining[identity]
		if !present {
			return nil, invariant("generic ABI binding identity is missing")
		}
		result = append(result, value)
		delete(remaining, identity)
	}
	if len(remaining) != 0 {
		return nil, invariant("generic ABI has foreign binding identities")
	}
	return result, nil
}

func method(
	owner *types.Func,
) (*types.Func, *types.Signature, error) {
	if owner == nil || owner.Origin() == nil {
		return nil, nil, invariant("generic method ABI owner is invalid")
	}
	owner = owner.Origin()
	signature, ok := owner.Type().(*types.Signature)
	if !ok ||
		signature.Recv() == nil ||
		signature.RecvTypeParams().Len() == 0 {
		return nil, nil, invariant("generic method ABI owner is not generic")
	}
	return owner, signature, nil
}

func invariant(reason string) error {
	return &api.InvariantError{
		Role:   api.RoleCallArgument,
		Reason: reason,
	}
}
