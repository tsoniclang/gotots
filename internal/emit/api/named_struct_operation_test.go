package api

import (
	"errors"
	"go/token"
	"go/types"
	"testing"
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
	if !ok ||
		requirement.Owner() != typeName ||
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
