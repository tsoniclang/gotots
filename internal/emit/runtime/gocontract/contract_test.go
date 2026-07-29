package gocontract_test

import (
	"context"
	"go/types"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/gocontract"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestContractBindsExactSelectedRuntimeTypes(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: filepath.Join(
			"..", "..", "..", "..",
			"testdata", "constructs", "control", "wave8",
		),
		Pattern: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	contract, err := gocontract.Resolve(program)
	if err != nil {
		t.Fatal(err)
	}
	runtimePackage := importedPackage(program.Roots()[0].Types(), "runtime")
	if runtimePackage == nil || !contract.Owns(runtimePackage) {
		t.Fatal("selected runtime package did not bind to the contract")
	}
	errorName := runtimePackage.Scope().Lookup("Error").(*types.TypeName)
	panicNilName := runtimePackage.Scope().Lookup("PanicNilError").(*types.TypeName)
	panicNil := panicNilName.Type()
	cases := []struct {
		source types.Type
		want   api.GoRuntimeType
	}{
		{types.Universe.Lookup("error").Type(), api.GoRuntimeTypeBuiltinError},
		{errorName.Type(), api.GoRuntimeTypeError},
		{panicNil, api.GoRuntimeTypePanicNilError},
		{types.NewPointer(panicNil), api.GoRuntimeTypePanicNilPointer},
		{types.Typ[types.String], api.GoRuntimeTypeInvalid},
	}
	for _, testCase := range cases {
		if got := contract.Classify(testCase.source); got != testCase.want {
			t.Fatalf("Classify(%s) = %d, want %d", testCase.source, got, testCase.want)
		}
	}
}

func importedPackage(root *types.Package, path string) *types.Package {
	seen := make(map[*types.Package]struct{})
	var visit func(*types.Package) *types.Package
	visit = func(current *types.Package) *types.Package {
		if current == nil {
			return nil
		}
		if _, visited := seen[current]; visited {
			return nil
		}
		seen[current] = struct{}{}
		if current.Path() == path {
			return current
		}
		for _, imported := range current.Imports() {
			if found := visit(imported); found != nil {
				return found
			}
		}
		return nil
	}
	return visit(root)
}
