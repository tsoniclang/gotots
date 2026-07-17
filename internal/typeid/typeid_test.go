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

// TestSignatureIgnoresReceiver: Go ignores the receiver when comparing
// signatures for identity, and abstract methods carry the enclosing
// interface as receiver. Structurally identical interfaces MUST share a
// canonical identity, and two methods with the same callable signature
// MUST share their signature identity (the receiver distinction lives in
// the method-declaration shape, not the callable signature).
func TestSignatureIgnoresReceiver(t *testing.T) {
	src := `package p
type A struct{}
type B struct{}
func (A) Read(x []byte) (int, error)  { return 0, nil }
func (*B) Read(x []byte) (int, error) { return 0, nil }
type I1 interface{ M() }
type I2 interface{ M() }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	aRead := methodByName(t, scope.Lookup("A").Type().(*types.Named), "Read").Type()
	bRead := methodByName(t, scope.Lookup("B").Type().(*types.Named), "Read").Type()
	if Canonical(aRead) != Canonical(bRead) {
		t.Errorf("same callable signature got distinct identities: %q vs %q", Canonical(aRead), Canonical(bRead))
	}
	if contains(Canonical(aRead), "recv(") {
		t.Errorf("receiver leaked into signature identity: %q", Canonical(aRead))
	}
	i1 := scope.Lookup("I1").Type().Underlying()
	i2 := scope.Lookup("I2").Type().Underlying()
	if Canonical(i1) != Canonical(i2) {
		t.Errorf("structurally identical interfaces got distinct identities: %q vs %q", Canonical(i1), Canonical(i2))
	}
}

// TestConstraintCompletenessAndAlphaEquivalence: a methodless type-set
// constraint must not collapse toward interface{}; differing tilde/union
// constraints must differ; alpha-equivalent binders must coincide.
func TestConstraintCompletenessAndAlphaEquivalence(t *testing.T) {
	src := `package p
func FInt32[T ~int32](x T)  { _ = x }
func FUint32[T ~uint32](x T) { _ = x }
func GT[T any](x T)  { _ = x }
func GU[U any](x U)  { _ = x }
func HMix[T ~int | ~uint | ~int64](x T) { _ = x }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	i32 := Canonical(scope.Lookup("FInt32").Type())
	u32 := Canonical(scope.Lookup("FUint32").Type())
	if i32 == u32 {
		t.Errorf("~int32 and ~uint32 constraints collapsed to the same identity: %q", i32)
	}
	if !contains(i32, "int32") {
		t.Errorf("constraint type set not serialized (collapsed toward interface{}): %q", i32)
	}
	gt := Canonical(scope.Lookup("GT").Type())
	gu := Canonical(scope.Lookup("GU").Type())
	if gt != gu {
		t.Errorf("alpha-equivalent generic signatures got distinct identities: %q vs %q", gt, gu)
	}
	hmix := Canonical(scope.Lookup("HMix").Type())
	if !contains(hmix, "int64") {
		t.Errorf("union constraint terms not fully serialized: %q", hmix)
	}
}

// TestEmbeddedFieldDistinct keeps the struct-embedding distinction.
func TestEmbeddedFieldDistinct(t *testing.T) {
	src := `package p
type T struct{ x int }
type Named struct{ F T }
type Embed struct{ T }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	if Canonical(scope.Lookup("Named").Type().Underlying()) == Canonical(scope.Lookup("Embed").Type().Underlying()) {
		t.Errorf("named-field and embedded-field structs share identity")
	}
}
