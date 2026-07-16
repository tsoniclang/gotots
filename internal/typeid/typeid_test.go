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

// TestReviewerIdentityDistinctions covers the round-11 reviewer's exact
// collision examples: each pair must have DISTINCT canonical identities.
func TestReviewerIdentityDistinctions(t *testing.T) {
	src := `package p
type T struct{ x int }
func (v T) M()  {}
func (v *T) N() {}
type Named struct{ F T }
type Embed struct{ T }
func FAny[A any](a A)               {}
func FComparable[A comparable](a A) {}
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	tType := scope.Lookup("T").Type().(*types.Named)
	mSig := methodByName(t, tType, "M").Type()
	nSig := methodByName(t, tType, "N").Type()
	if Canonical(mSig) == Canonical(nSig) {
		t.Errorf("value- and pointer-receiver methods share identity: %s", Canonical(mSig))
	}
	if !contains(Canonical(nSig), "recv(*") {
		t.Errorf("pointer receiver not represented: %s", Canonical(nSig))
	}
	named := scope.Lookup("Named").Type().Underlying()
	embed := scope.Lookup("Embed").Type().Underlying()
	if Canonical(named) == Canonical(embed) {
		t.Errorf("named-field and embedded-field structs share identity: %s", Canonical(named))
	}
	anySig := scope.Lookup("FAny").Type()
	cmpSig := scope.Lookup("FComparable").Type()
	if Canonical(anySig) == Canonical(cmpSig) {
		t.Errorf("generic funcs with any vs comparable constraint share identity: %s", Canonical(anySig))
	}
}

func methodByName(t *testing.T, named *types.Named, name string) *types.Func {
	t.Helper()
	for i := range named.NumMethods() {
		if named.Method(i).Name() == name {
			return named.Method(i)
		}
	}
	t.Fatalf("method %s not found", name)
	return nil
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestHasUnsupported verifies the fail-closed detector: a poisoned
// identity is flagged, an ordinary one is not.
func TestHasUnsupported(t *testing.T) {
	src := `package p
type T struct{ x int }
func F(a int) string { return "" }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	for _, name := range []string{"T", "F"} {
		id := Canonical(scope.Lookup(name).Type())
		if HasUnsupported(id) {
			t.Errorf("%s: ordinary type flagged unsupported: %q", name, id)
		}
	}
	if !HasUnsupported("x" + unsupportedMarker + "y") {
		t.Errorf("poisoned identity not detected")
	}
}
