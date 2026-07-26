package api

import "go/types"

type DeclarationRequirementKind uint8

const (
	DeclarationRequirementInvalid DeclarationRequirementKind = iota
	DeclarationRequirementNamedStructCompanion
)

type DeclarationRequirement struct {
	owner     types.Object
	kind      DeclarationRequirementKind
	companion CompanionOperation
}

func NewNamedStructCompanionRequirement(
	typeName *types.TypeName,
	operation CompanionOperation,
) (DeclarationRequirement, error) {
	switch {
	case typeName == nil:
		return DeclarationRequirement{}, &PlacementRequestError{
			Reason: "companion type is nil",
		}
	case !operation.Valid():
		return DeclarationRequirement{}, &PlacementRequestError{
			Reason: "companion operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     typeName,
		kind:      DeclarationRequirementNamedStructCompanion,
		companion: operation,
	}, nil
}

func (r DeclarationRequirement) Valid() bool {
	if r.kind != DeclarationRequirementNamedStructCompanion ||
		!r.companion.Valid() {
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

func (r DeclarationRequirement) NamedStructCompanion() (
	*types.TypeName,
	CompanionOperation,
	bool,
) {
	if !r.Valid() {
		return nil, CompanionInvalid, false
	}
	typeName, ok := r.owner.(*types.TypeName)
	return typeName, r.companion, ok
}
