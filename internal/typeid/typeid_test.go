package typeid

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func pkgOf(t *testing.T, path, src string) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(path, fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

// TestUnexportedStructuralIdentityDistinct: two structurally
// spelled-alike interfaces with unexported methods from different
// packages must have DISTINCT canonical identities.
func TestUnexportedStructuralIdentityDistinct(t *testing.T) {
	a := pkgOf(t, "example.com/a", "package a\ntype S interface{ tag() int }")
	b := pkgOf(t, "example.com/b", "package b\ntype S interface{ tag() int }")
	ida := Canonical(a.Scope().Lookup("S").Type().Underlying())
	idb := Canonical(b.Scope().Lookup("S").Type().Underlying())
	if ida == idb {
		t.Fatalf("distinct unexported interfaces collided: %s", ida)
	}
}

// TestAliasResolvesToTarget: an alias is not a distinct dynamic type.
func TestAliasResolvesToTarget(t *testing.T) {
	p := pkgOf(t, "example.com/c", "package c\ntype A = []int\ntype B = A")
	idA := Canonical(p.Scope().Lookup("A").Type())
	idB := Canonical(p.Scope().Lookup("B").Type())
	if idA != idB || idA != "[]int" {
		t.Fatalf("aliases did not normalize: %q vs %q", idA, idB)
	}
}

// TestPathQualification: same package NAME, different paths → distinct.
func TestPathQualification(t *testing.T) {
	a := pkgOf(t, "one/util", "package util\ntype T struct{ X int }")
	b := pkgOf(t, "two/util", "package util\ntype T struct{ X int }")
	if Canonical(a.Scope().Lookup("T").Type()) == Canonical(b.Scope().Lookup("T").Type()) {
		t.Fatal("same-name packages collided")
	}
}
