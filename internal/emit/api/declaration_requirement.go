package api

import "go/types"

type DeclarationRequirementKind uint8

const (
	DeclarationRequirementInvalid DeclarationRequirementKind = iota
	DeclarationRequirementNamedStructOperation
)

type DeclarationRequirement struct {
	owner     types.Object
	kind      DeclarationRequirementKind
	operation NamedStructOperation
}

func NewNamedStructOperationRequirement(
	typeName *types.TypeName,
	operation NamedStructOperation,
) (DeclarationRequirement, error) {
	switch {
	case typeName == nil:
		return DeclarationRequirement{}, &PlacementRequestError{
			Reason: "named-struct operation type is nil",
		}
	case !operation.Valid():
		return DeclarationRequirement{}, &PlacementRequestError{
			Reason: "named-struct operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     typeName,
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
	}, nil
}

func (r DeclarationRequirement) Valid() bool {
	if r.kind != DeclarationRequirementNamedStructOperation ||
		!r.operation.Valid() {
		return false
	}
	_, ok := r.owner.(*types.TypeName)
	return ok
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
