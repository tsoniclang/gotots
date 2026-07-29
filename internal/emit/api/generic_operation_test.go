package api

import (
	"go/token"
	"go/types"
	"regexp"
	"testing"
)

func TestGenericOperationIdentifiersAreTotalUniqueTargetIdentifiers(t *testing.T) {
	targetIdentifier := regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	seen := make(map[string]GenericOperation)
	for operation := GenericOperationZero; operation <= GenericOperationConstraintMethod; operation++ {
		identifier := operation.Identifier()
		if !targetIdentifier.MatchString(identifier) {
			t.Fatalf(
				"generic operation %d identifier = %q",
				operation,
				identifier,
			)
		}
		if previous, duplicate := seen[identifier]; duplicate {
			t.Fatalf(
				"generic operations %d and %d share identifier %q",
				previous,
				operation,
				identifier,
			)
		}
		seen[identifier] = operation
	}
	if GenericOperationInvalid.Identifier() != "" ||
		GenericOperation(GenericOperationConstraintMethod+1).Identifier() != "" {
		t.Fatal("invalid generic operation has a target identifier")
	}
}

func TestConstraintMethodSelectionPreservesExactMethodIdentity(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first", "first")
	secondPackage := types.NewPackage("example.com/second", "second")
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]),
		),
		false,
	)
	selectionKey := func(method *types.Func) string {
		t.Helper()
		selection, err := SelectGenericConstraintMethod(method)
		if err != nil {
			t.Fatal(err)
		}
		key, err := selection.IdentityPrefix()
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	readFirst := types.NewFunc(
		token.NoPos,
		firstPackage,
		"Read",
		signature,
	)
	readSecond := types.NewFunc(
		token.NoPos,
		secondPackage,
		"Read",
		signature,
	)
	write := types.NewFunc(
		token.NoPos,
		firstPackage,
		"Write",
		signature,
	)
	privateFirst := types.NewFunc(
		token.NoPos,
		firstPackage,
		"read",
		signature,
	)
	privateSecond := types.NewFunc(
		token.NoPos,
		secondPackage,
		"read",
		signature,
	)
	if selectionKey(readFirst) != selectionKey(readSecond) {
		t.Fatal("equivalent exported constraint methods differ by package")
	}
	if selectionKey(readFirst) == selectionKey(write) {
		t.Fatal("different constraint methods with one signature share identity")
	}
	if selectionKey(privateFirst) == selectionKey(privateSecond) {
		t.Fatal("unexported constraint methods from different packages share identity")
	}
	if _, err := SelectGenericOperation(
		GenericOperationConstraintMethod,
	); err == nil {
		t.Fatal("constraint method accepted without exact method evidence")
	}
}

func TestGenericOperationContractCarriesCrossParameterSignature(t *testing.T) {
	constraint := types.NewInterfaceType(nil, nil).Complete()
	left := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "T", nil),
		constraint,
	)
	right := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "U", nil),
		constraint,
	)
	operationSignature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", left),
			types.NewVar(token.NoPos, nil, "count", right),
		),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", left)),
		false,
	)
	ownerSignature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{left, right},
		operationSignature.Params(),
		operationSignature.Results(),
		false,
	)
	owner := types.NewFunc(token.NoPos, nil, "Shift", ownerSignature)
	selection, err := SelectGenericOperation(
		GenericOperationBinaryShiftLeft,
	)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := NewGenericOperationContract(
		owner,
		"binary_shift_left|(T,U)->T",
		"$go$binary_shift_left",
		selection,
		operationSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	callable, err := NewGenericCallable(
		owner,
		[]*types.TypeParam{left, right},
		[]*GenericOperationContract{contract},
	)
	if err != nil {
		t.Fatal(err)
	}
	operations := callable.Operations()
	if len(operations) != 1 ||
		!types.Identical(operations[0].Signature(), operationSignature) ||
		operations[0].Operation() != GenericOperationBinaryShiftLeft {
		t.Fatalf("cross-parameter generic operation = %#v", operations)
	}
	foreignParameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "V", nil),
		constraint,
	)
	foreignOwner := types.NewFunc(
		token.NoPos,
		nil,
		"Foreign",
		types.NewSignatureType(
			nil,
			nil,
			[]*types.TypeParam{foreignParameter},
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "value", foreignParameter),
			),
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", foreignParameter),
			),
			false,
		),
	)
	if _, err := NewGenericOperationRequirement(
		foreignOwner,
		contract,
	); err == nil {
		t.Fatal("foreign generic owner accepted another callable's operation")
	}
	foreignOperation := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", foreignParameter),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", foreignParameter),
		),
		false,
	)
	if _, err := NewGenericOperationContract(
		owner,
		"copy|(V)->V",
		"$go$copy",
		mustGenericOperation(t, GenericOperationCopy),
		foreignOperation,
	); err == nil {
		t.Fatal("generic operation accepted a foreign type parameter")
	}
}

func mustGenericOperation(
	t *testing.T,
	operation GenericOperation,
) GenericOperationSelection {
	t.Helper()
	selection, err := SelectGenericOperation(operation)
	if err != nil {
		t.Fatal(err)
	}
	return selection
}
