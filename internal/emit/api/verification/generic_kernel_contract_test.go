package api_test

import (
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGenericKernelSelectionSeparatesOperationsFromRepresentation(t *testing.T) {
	owner, parameter, _ := genericRepresentationOwner("Kernel")
	representation, err := NewGenericRepresentationRequirement(
		owner,
		parameter,
		GenericRepresentationPointer,
	)
	if err != nil {
		t.Fatal(err)
	}
	required, err := GenericKernelRequired(
		owner,
		[]DeclarationRequirement{representation},
	)
	if err != nil {
		t.Fatal(err)
	}
	if required {
		t.Fatal("representation-only demand selected a runtime kernel")
	}

	operation := genericCopyOperation(t, owner, parameter)
	operationRequirement, err := NewGenericOperationRequirement(owner, operation)
	if err != nil {
		t.Fatal(err)
	}
	required, err = GenericKernelRequired(
		owner,
		[]DeclarationRequirement{representation, operationRequirement},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !required {
		t.Fatal("generic operation did not select its runtime kernel")
	}
}

func TestGenericKernelSelectionRejectsForeignAndNonGenericOwners(t *testing.T) {
	owner, parameter, _ := genericRepresentationOwner("Owner")
	foreign, _, _ := genericRepresentationOwner("ForeignKernel")
	representation, err := NewGenericRepresentationRequirement(
		owner,
		parameter,
		GenericRepresentationStorage,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenericKernelRequired(
		foreign,
		[]DeclarationRequirement{representation},
	); err == nil {
		t.Fatal("foreign generic owner accepted another declaration's representation")
	}
	nonGeneric := types.NewFunc(
		token.NoPos,
		types.NewPackage("example.com/plain", "plain"),
		"Plain",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(),
			types.NewTuple(),
			false,
		),
	)
	if _, err := GenericKernelRequired(nonGeneric, nil); err == nil {
		t.Fatal("non-generic declaration was accepted as a generic kernel owner")
	}
}

func genericCopyOperation(
	t *testing.T,
	owner *types.Func,
	parameter *types.TypeParam,
) *GenericOperationContract {
	t.Helper()
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "value", parameter)),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", parameter)),
		false,
	)
	selection, err := SelectGenericOperation(GenericOperationCopy)
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewGenericOperationContract(
		owner,
		"copy|(T)->T",
		"$go$copy",
		GenericFunctionOperationConsumer(),
		selection,
		signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	return operation
}
