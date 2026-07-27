package api

import (
	"go/token"
	"go/types"
	"testing"
)

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
	context := (Context{}).WithAddressableStorage(owner, selections)
	selections[first] = "mutated"
	selections[second] = "forged"

	if context.ArtifactOwner() != owner {
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
