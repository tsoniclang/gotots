package api_test

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestNamedStructOperationRequestCarriesTypedDeclarationRequirement(t *testing.T) {
	typeName := types.NewTypeName(
		token.NoPos,
		types.NewPackage("example.com/records", "records"),
		"Box",
		nil,
	)
	request, err := NewNamedStructOperationRequest(typeName, NamedStructOperationCopy)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	sourceOwner, sourceOwned := requirement.Owner().Source()
	if !ok ||
		!sourceOwned ||
		sourceOwner != typeName ||
		requirement.Kind() != DeclarationRequirementNamedStructOperation {
		t.Fatalf("declaration requirement = %#v, %t", requirement, ok)
	}
	owner, operation, ok := requirement.NamedStructOperation()
	if !ok || owner != typeName || operation != NamedStructOperationCopy {
		t.Fatalf(
			"named-struct operation = %v, %v, %t",
			owner,
			operation,
			ok,
		)
	}
	if request.LegalScope() != ScopeOwningFile ||
		request.PreferredScope() != ScopeOwningFile ||
		request.Execution() != ExecutionStatic {
		t.Fatalf(
			"placement = legal %d, preferred %d, execution %d",
			request.LegalScope(),
			request.PreferredScope(),
			request.Execution(),
		)
	}
}

func TestNamedStructOperationRequestRejectsInvalidOwners(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		typeName  *types.TypeName
		operation NamedStructOperation
	}{
		{name: "nil type", operation: NamedStructOperationCopy},
		{
			name: "invalid operation",
			typeName: types.NewTypeName(
				token.NoPos,
				types.NewPackage("example.com/records", "records"),
				"Box",
				nil,
			),
			operation: NamedStructOperationInvalid,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewNamedStructOperationRequest(
				testCase.typeName,
				testCase.operation,
			)
			var requestError *RootRequestError
			if !errors.As(err, &requestError) {
				t.Fatalf("error = %#v, want RootRequestError", err)
			}
		})
	}
}

func TestNamedStructOperationMemberNamesAreClosed(t *testing.T) {
	for _, testCase := range []struct {
		operation NamedStructOperation
		want      string
	}{
		{operation: NamedStructOperationZero, want: "$zero"},
		{operation: NamedStructOperationCopy, want: "$copy"},
		{operation: NamedStructOperationEqual, want: "$equal"},
		{operation: NamedStructOperationHash, want: "$hash"},
		{operation: NamedStructOperationConvert, want: "$convert"},
	} {
		got, err := NamedStructOperationMemberName(testCase.operation)
		if err != nil {
			t.Fatal(err)
		}
		if got != testCase.want {
			t.Fatalf("member name = %q, want %q", got, testCase.want)
		}
	}
}

func TestNamedStructOperationsHaveTotalGenericConsumerIdentities(t *testing.T) {
	seen := make(map[GenericOperationConsumer]NamedStructOperation)
	for operation := NamedStructOperationZero; operation <= NamedStructOperationAssign; operation++ {
		consumer, err := GenericNamedStructOperationConsumer(operation)
		if err != nil {
			t.Fatalf("%s: %v", operation, err)
		}
		if previous, duplicate := seen[consumer]; duplicate {
			t.Fatalf(
				"%s and %s share generic consumer %d",
				previous,
				operation,
				consumer,
			)
		}
		seen[consumer] = operation
		selected, ok := consumer.NamedStructOperation()
		if !consumer.Valid() || !ok || selected != operation {
			t.Fatalf(
				"%s generic consumer round trip = %s, %t",
				operation,
				selected,
				ok,
			)
		}
		if consumer.Identity() != "named-struct-"+operation.String() {
			t.Fatalf(
				"%s generic consumer identity = %q",
				operation,
				consumer.Identity(),
			)
		}
	}
	if _, err := GenericNamedStructOperationConsumer(
		NamedStructOperationInvalid,
	); err == nil {
		t.Fatal("invalid named-struct operation acquired a generic consumer")
	}
}
