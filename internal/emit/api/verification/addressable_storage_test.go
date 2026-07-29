package api_test

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestAddressableStorageRequestCarriesExactFunctionAndVariableIdentity(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/storage", "storage")
	owner := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Use",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	variable := types.NewVar(token.Pos(20), sourcePackage, "value", types.Typ[types.Int32])
	request, err := NewAddressableStorageRequest(owner, variable)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	sourceOwner, sourceOwned := requirement.Owner().Source()
	if !ok ||
		!sourceOwned ||
		sourceOwner != owner ||
		requirement.Kind() != DeclarationRequirementAddressableStorage {
		t.Fatalf("declaration requirement = %#v, %t", requirement, ok)
	}
	gotOwner, gotVariable, ok := requirement.AddressableStorage()
	if !ok || gotOwner != owner || gotVariable != variable {
		t.Fatalf(
			"addressable storage = %v, %v, %t",
			gotOwner,
			gotVariable,
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

func TestAddressableStorageRequirementRejectsInvalidOwners(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/storage", "storage")
	owner := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Use",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	variable := types.NewVar(token.Pos(20), sourcePackage, "value", types.Typ[types.Int32])
	field := types.NewField(
		token.Pos(30),
		sourcePackage,
		"Value",
		types.Typ[types.Int32],
		false,
	)
	for _, testCase := range []struct {
		name     string
		owner    *types.Func
		variable *types.Var
	}{
		{name: "nil owner", variable: variable},
		{name: "nil variable", owner: owner},
		{name: "field", owner: owner, variable: field},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewAddressableStorageRequest(
				testCase.owner,
				testCase.variable,
			)
			var requestError *RootRequestError
			if !errors.As(err, &requestError) {
				t.Fatalf("error = %#v, want RootRequestError", err)
			}
		})
	}
}

func TestAddressableStorageRequirementNeverKeysBySpelling(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/storage", "storage")
	owner := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Use",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	first := types.NewVar(token.Pos(20), sourcePackage, "value", types.Typ[types.Int32])
	second := types.NewVar(token.Pos(30), sourcePackage, "value", types.Typ[types.Int32])
	firstRequirement, err := NewAddressableStorageRequirement(owner, first)
	if err != nil {
		t.Fatal(err)
	}
	secondRequirement, err := NewAddressableStorageRequirement(owner, second)
	if err != nil {
		t.Fatal(err)
	}
	if firstRequirement == secondRequirement {
		t.Fatal("same-spelling variables collapsed to one storage requirement")
	}
}

func TestDeclarationRequirementKindIDsArePinned(t *testing.T) {
	if DeclarationRequirementNamedStructOperation != 1 ||
		DeclarationRequirementAddressableStorage != 2 ||
		DeclarationRequirementConstantProjection != 3 ||
		DeclarationRequirementLocalConstantProjection != 4 ||
		DeclarationRequirementGenericOperation != 5 ||
		DeclarationRequirementAnonymousStruct != 6 ||
		DeclarationRequirementMapSpecialization != 7 ||
		DeclarationRequirementInterfaceAdapter != 8 ||
		DeclarationRequirementAnonymousInterface != 9 ||
		DeclarationRequirementInterfaceMethodToken != 10 ||
		DeclarationRequirementInterfaceDynamicTypeToken != 11 ||
		DeclarationRequirementGenericCapability != 12 ||
		DeclarationRequirementCallableControl != 13 ||
		DeclarationRequirementCooperativeCallable != 14 ||
		DeclarationRequirementCallableABI != 15 ||
		DeclarationRequirementKind(16).Valid() {
		t.Fatal("declaration requirement kind IDs drifted")
	}
}

func TestAddressableStorageContextCopiesAndKeysExactVariables(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/storage", "storage")
	owner := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Use",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	first := types.NewVar(token.Pos(20), sourcePackage, "value", types.Typ[types.Int32])
	second := types.NewVar(token.Pos(30), sourcePackage, "value", types.Typ[types.Int32])
	selections := map[*types.Var]string{first: "value$storage"}
	context, err := (Context{}).WithSourceArtifactOwner(
		MustSourceArtifactOwner(owner),
	)
	if err != nil {
		t.Fatal(err)
	}
	context, err = context.WithAddressableStorage(owner, selections)
	if err != nil {
		t.Fatal(err)
	}
	selections[first] = "mutated"
	selections[second] = "forged"

	if context.ArtifactOwner() != MustSourceArtifactOwner(owner) {
		t.Fatal("addressable storage lost its exact artifact owner")
	}
	if name, ok := context.AddressableStorageName(first); !ok ||
		name != "value$storage" {
		t.Fatalf("selected storage = %q, %t", name, ok)
	}
	if _, ok := context.AddressableStorageName(second); ok {
		t.Fatal("same-spelling unselected variable acquired storage")
	}
}

func TestFunctionCapabilityRejectsDivergentCurrentArtifactOwner(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/storage", "storage")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	owner := types.NewFunc(token.Pos(10), sourcePackage, "Use", signature)
	foreign := types.NewFunc(token.Pos(10), sourcePackage, "Use", signature)
	context, err := (Context{}).WithSourceArtifactOwner(
		MustSourceArtifactOwner(owner),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = context.WithAddressableStorage(foreign, nil)
	var contextError *ContextError
	if !errors.As(err, &contextError) {
		t.Fatalf("divergent function owner error = %#v, want ContextError", err)
	}
}
