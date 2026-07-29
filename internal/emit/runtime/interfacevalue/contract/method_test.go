package contract

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"
)

func TestRuntimeErrorMethodContractJoinsSelectedToolchain(t *testing.T) {
	runtimePackage, err := importer.Default().Import("runtime")
	if err != nil {
		t.Fatal(err)
	}
	errorName, ok := runtimePackage.Scope().Lookup("Error").(*types.TypeName)
	if !ok {
		t.Fatal("selected runtime package has no Error type")
	}
	errorInterface, ok := errorName.Type().Underlying().(*types.Interface)
	if !ok {
		t.Fatal("selected runtime.Error is not an interface")
	}
	errorInterface = errorInterface.Complete()
	found := false
	for index := range errorInterface.NumMethods() {
		if Method(errorInterface.Method(index)) == MethodRuntimeError {
			found = true
		}
	}
	if !found {
		t.Fatal("selected runtime.Error does not join the closed method contract")
	}
}

func TestRuntimeMethodContractUsesExactGoMethodIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/user", "user")
	sameIdentity := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"RuntimeError",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	differentSignature := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"RuntimeError",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(types.NewVar(
				token.NoPos,
				sourcePackage,
				"value",
				types.Typ[types.Int],
			)),
			nil,
			false,
		),
	)
	foreignSpelling := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"LooksLikeRuntimeError",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	if Method(sameIdentity) != MethodRuntimeError {
		t.Fatal("equivalent exported method did not share runtime.Error identity")
	}
	if Method(differentSignature) != MethodInvalid {
		t.Fatal("same spelling with a foreign signature acquired a runtime token")
	}
	if Method(foreignSpelling) != MethodInvalid {
		t.Fatal("foreign spelling acquired a runtime token")
	}
}
