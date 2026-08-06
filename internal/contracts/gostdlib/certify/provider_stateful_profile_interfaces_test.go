package certify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

func TestRetainedCallableContributesTypedInterfaceRoots(t *testing.T) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "state.go", `package state
		type Failure interface { Error() string }
		type State struct {
			invoke func(Failure) Failure
		}
	`, 0)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := new(types.Config).Check(
		"example.com/state",
		fileSet,
		[]*ast.File{file},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	state, ok := selected.Scope().Lookup("State").Type().(*types.Named)
	if !ok {
		t.Fatal("State is not a named type")
	}
	failure, ok := selected.Scope().Lookup("Failure").(*types.TypeName)
	if !ok {
		t.Fatal("Failure is not a type name")
	}
	contract, err := environmentcontract.Describe(failure)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := retainedProviderInterfaces(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(retained) != 1 || retained[contract.Identity()] == nil {
		t.Fatalf("retained interfaces = %#v, want only %q", retained, contract.Identity())
	}
}
