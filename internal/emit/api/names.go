package api

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type TemporaryKind uint8

const (
	TemporaryInvalid TemporaryKind = iota
	TemporaryAssignmentValue
	TemporaryMultipleResults
	TemporaryCompositeField
	TemporaryReceiverValue
	TemporaryCallArgument
	TemporaryCallCallee
	TemporaryMapOperand
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

type PackageVariableReference struct {
	stateName string
	fieldName string
	requests  []PlacementRequest
}

func NewPackageVariableReference(
	stateName string,
	fieldName string,
	requests ...PlacementRequest,
) (PackageVariableReference, error) {
	switch {
	case stateName == "":
		return PackageVariableReference{},
			&NameError{Reason: "package state name is empty"}
	case fieldName == "":
		return PackageVariableReference{},
			&NameError{Reason: "package variable field name is empty"}
	}
	return PackageVariableReference{
		stateName: stateName,
		fieldName: fieldName,
		requests:  slices.Clone(requests),
	}, nil
}

func (r PackageVariableReference) StateName() string {
	return r.stateName
}

func (r PackageVariableReference) FieldName() string {
	return r.fieldName
}

func (r PackageVariableReference) Requests() []PlacementRequest {
	return slices.Clone(r.requests)
}

func (r PackageVariableReference) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.Identifier(r.stateName),
		nil,
		factory.Identifier(r.fieldName),
		tsgo.NodeFlagsNone,
	)
}

type Names interface {
	Declare(types.Object) (string, error)
	Parameter(*types.Var, int) (string, error)
	Reference(types.Object) (NameReference, error)
	TypeReference(types.Object) (NameReference, error)
	PackageVariable(*types.Var) (PackageVariableReference, error)
	Companion(*types.TypeName, CompanionOperation) (NameReference, error)
	Member(*types.Var) (string, error)
	Primitive(PrimitiveAlias) (NameReference, error)
	Runtime(RuntimeSymbol, ImportPhase) (NameReference, error)
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
	case TemporaryCallCallee:
		return "__gotots_callee_", nil
	case TemporaryMapOperand:
		return "__gotots_map_", nil
	default:
		return "", &NameError{
			Reason: fmt.Sprintf("temporary kind %d is invalid", kind),
		}
	}
}
