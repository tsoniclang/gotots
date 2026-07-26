package api

import (
	"errors"
	"go/token"
	"go/types"
	"testing"
)

func TestCompanionRequestCarriesTypedOwningFileIdentity(t *testing.T) {
	typeName := types.NewTypeName(
		token.NoPos,
		types.NewPackage("example.com/records", "records"),
		"Box",
		nil,
	)
	request, err := NewCompanionRequest(typeName, CompanionCopy)
	if err != nil {
		t.Fatal(err)
	}
	owner, ok := request.Companion()
	if !ok ||
		owner.TypeName() != typeName ||
		owner.Operation() != CompanionCopy {
		t.Fatalf("companion owner = %#v, %t", owner, ok)
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

func TestCompanionRequestRejectsInvalidOwners(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		typeName  *types.TypeName
		operation CompanionOperation
	}{
		{name: "nil type", operation: CompanionCopy},
		{
			name: "invalid operation",
			typeName: types.NewTypeName(
				token.NoPos,
				types.NewPackage("example.com/records", "records"),
				"Box",
				nil,
			),
			operation: CompanionInvalid,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewCompanionRequest(
				testCase.typeName,
				testCase.operation,
			)
			var requestError *PlacementRequestError
			if !errors.As(err, &requestError) {
				t.Fatalf("error = %#v, want PlacementRequestError", err)
			}
		})
	}
}
