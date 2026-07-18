// A generic type alias's DECLARATION evidence records its own type
// parameters and constraints (the declaration domain), distinct from its
// transparent Target (the use-site domain). Two generic aliases whose
// instantiated targets coincide but whose declarations differ in
// constraints are distinguishable in the declaration evidence.
package census

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/typeid"
)

func aliasShapeOf(t *testing.T, src, name string) AliasShape {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("p", fset, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}
	obj := pkg.Scope().Lookup(name).(*types.TypeName)
	alias := obj.Type().(*types.Alias)
	var terr error
	binders := typeid.AliasBinders(alias)
	target, err := typeid.Canonical(obj.Type())
	if err != nil {
		t.Fatal(err)
	}
	shape := AliasShape{ID: "p." + name, Target: target, TypeParams: typeParamShapesIn(alias.TypeParams(), binders, &terr)}
	if terr != nil {
		t.Fatal(terr)
	}
	return shape
}

func TestGenericAliasDeclarationEvidence(t *testing.T) {
	comparable1 := aliasShapeOf(t, "package p\ntype A[T comparable] = func(int) T", "A")
	anyConstraint := aliasShapeOf(t, "package p\ntype C[V any] = func(int) V", "C")

	// Same transparent target (the use-site domain coincides)...
	if comparable1.Target != anyConstraint.Target {
		t.Fatalf("expected identical transparent targets, got %q vs %q", comparable1.Target, anyConstraint.Target)
	}
	// ...but the DECLARATION domain distinguishes the constraint.
	if len(comparable1.TypeParams) != 1 || comparable1.TypeParams[0].Constraint != "comparable" {
		t.Fatalf("A declaration evidence missing comparable constraint: %+v", comparable1.TypeParams)
	}
	if len(anyConstraint.TypeParams) != 1 || anyConstraint.TypeParams[0].Constraint == "comparable" {
		t.Fatalf("C declaration evidence must not carry the comparable constraint: %+v", anyConstraint.TypeParams)
	}
}
