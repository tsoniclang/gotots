package api

import "go/types"

type DeclarationRequirementKind uint8

const (
	DeclarationRequirementInvalid DeclarationRequirementKind = iota
	DeclarationRequirementNamedStructOperation
	DeclarationRequirementAddressableStorage
)

func (k DeclarationRequirementKind) Valid() bool {
	return k == DeclarationRequirementNamedStructOperation ||
		k == DeclarationRequirementAddressableStorage
}

type DeclarationRequirement struct {
	owner     types.Object
	kind      DeclarationRequirementKind
	operation NamedStructOperation
	variable  *types.Var
}

func NewNamedStructOperationRequirement(
	typeName *types.TypeName,
	operation NamedStructOperation,
) (DeclarationRequirement, error) {
	switch {
	case typeName == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "named-struct operation type is nil",
		}
	case !operation.Valid():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "named-struct operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     typeName,
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
	}, nil
}

func NewAddressableStorageRequirement(
	owner *types.Func,
	variable *types.Var,
) (DeclarationRequirement, error) {
	switch {
	case owner == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage owner is nil",
		}
	case variable == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage variable is nil",
		}
	case variable.IsField():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "addressable-storage variable is a field",
		}
	}
	return DeclarationRequirement{
		owner:    owner,
		kind:     DeclarationRequirementAddressableStorage,
		variable: variable,
	}, nil
}

func (r DeclarationRequirement) Valid() bool {
	if !r.kind.Valid() {
		return false
	}
	switch r.kind {
	case DeclarationRequirementNamedStructOperation:
		if !r.operation.Valid() || r.variable != nil {
			return false
		}
		_, ok := r.owner.(*types.TypeName)
		return ok
	case DeclarationRequirementAddressableStorage:
		if r.operation != NamedStructOperationInvalid ||
			r.variable == nil ||
			r.variable.IsField() {
			return false
		}
		_, ok := r.owner.(*types.Func)
		return ok
	default:
		return false
	}
}

func (r DeclarationRequirement) Owner() types.Object {
	return r.owner
}

func (r DeclarationRequirement) Kind() DeclarationRequirementKind {
	return r.kind
}

func (r DeclarationRequirement) NamedStructOperation() (
	*types.TypeName,
	NamedStructOperation,
	bool,
) {
	if !r.Valid() {
		return nil, NamedStructOperationInvalid, false
	}
	typeName, ok := r.owner.(*types.TypeName)
	return typeName, r.operation, ok
}

func (r DeclarationRequirement) AddressableStorage() (
	*types.Func,
	*types.Var,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementAddressableStorage {
		return nil, nil, false
	}
	owner, ok := r.owner.(*types.Func)
	return owner, r.variable, ok
}
