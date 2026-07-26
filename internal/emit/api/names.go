package api

import (
	"fmt"
	"go/types"
	"slices"
)

type TemporaryKind uint8

const (
	TemporaryInvalid TemporaryKind = iota
	TemporaryAssignmentValue
	TemporaryMultipleResults
	TemporaryCompositeField
	TemporaryReceiverValue
	TemporaryCallArgument
)

type NameReference struct {
	name     string
	requests []PlacementRequest
}

func NewNameReference(name string, requests ...PlacementRequest) (NameReference, error) {
	if name == "" {
		return NameReference{}, &NameError{Reason: "reference name is empty"}
	}
	return NameReference{name: name, requests: slices.Clone(requests)}, nil
}

func (r NameReference) Name() string {
	return r.name
}

func (r NameReference) Requests() []PlacementRequest {
	return slices.Clone(r.requests)
}

type Names interface {
	Declare(types.Object) (string, error)
	Reference(types.Object) (NameReference, error)
	Companion(*types.TypeName, CompanionOperation) (NameReference, error)
	Member(*types.Var) (string, error)
	Primitive(PrimitiveAlias) (NameReference, error)
	Temporary(TemporaryKind) (string, error)
	ModuleExport(types.Object) (bool, error)
}

func TemporaryPrefix(kind TemporaryKind) (string, error) {
	switch kind {
	case TemporaryAssignmentValue:
		return "__gotots_assign_", nil
	case TemporaryMultipleResults:
		return "__gotots_results_", nil
	case TemporaryCompositeField:
		return "__gotots_field_", nil
	case TemporaryReceiverValue:
		return "__gotots_receiver_", nil
	case TemporaryCallArgument:
		return "__gotots_argument_", nil
	default:
		return "", &NameError{
			Reason: fmt.Sprintf("temporary kind %d is invalid", kind),
		}
	}
}
