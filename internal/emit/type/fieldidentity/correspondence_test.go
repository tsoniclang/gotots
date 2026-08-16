package fieldidentity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestGenericFieldCorrespondenceUsesSelectedIdentityAndOrdinal(t *testing.T) {
	checked := checkedPackage(t, `package fields

type Pair[T, U any] struct {
	First func(T) U
	Second func(U) T
}

type Other[T any] struct {
	Second func(T) T
}
`)
	pair := checked.Scope().Lookup("Pair").Type().(*types.Named)
	selectedType, err := types.Instantiate(
		types.NewContext(),
		pair,
		[]types.Type{types.Typ[types.Int32], types.Typ[types.String]},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	selected := selectedType.(*types.Named)
	selectedStruct := selected.Underlying().(*types.Struct)
	correspondence, resolved, err := Resolve(
		selected,
		selectedStruct.Field(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved {
		t.Fatal("generic selected field was not resolved")
	}
	owner, declaration, occurrence := correspondence.Parts()
	if owner != pair.Obj() {
		t.Fatal("field correspondence lost its nominal origin")
	}
	if correspondence.DeclarationField() != pair.Underlying().(*types.Struct).Field(1) {
		t.Fatal("field correspondence lost its declaration object")
	}
	if got := types.TypeString(declaration, nil); got != "func(U) T" {
		t.Fatalf("declaration field type = %s", got)
	}
	if got := types.TypeString(occurrence, nil); got != "func(string) int32" {
		t.Fatalf("selected field type = %s", got)
	}

	other := checked.Scope().Lookup("Other").Type().(*types.Named)
	otherType, err := types.Instantiate(
		types.NewContext(),
		other,
		[]types.Type{types.Typ[types.Int32]},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	foreign := otherType.(*types.Named).Underlying().(*types.Struct).Field(0)
	if _, _, err := Resolve(selected, foreign); err == nil {
		t.Fatal("same-spelling foreign field was accepted")
	}
}

func TestFieldCorrespondenceIgnoresNonGenericStructs(t *testing.T) {
	checked := checkedPackage(t, `package fields

type Plain struct {
	Apply func(int32) int32
}
`)
	plain := checked.Scope().Lookup("Plain").Type().(*types.Named)
	field := plain.Underlying().(*types.Struct).Field(0)
	if _, resolved, err := Resolve(plain, field); err != nil || resolved {
		t.Fatalf("non-generic field resolved=%v err=%v", resolved, err)
	}
	var absent *types.Named
	if _, resolved, err := Resolve(absent, nil); err != nil || resolved {
		t.Fatalf("typed-nil field resolved=%v err=%v", resolved, err)
	}
}

func checkedPackage(t *testing.T, sourceText string) *types.Package {
	t.Helper()
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(
		fileSet,
		"source.go",
		sourceText,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := new(types.Config).Check(
		"example.com/fields",
		fileSet,
		[]*ast.File{source},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return checked
}
