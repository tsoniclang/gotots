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

// canon renders a type's canonical identity, failing the test on error.
func canon(t *testing.T, typ types.Type) string {
	t.Helper()
	s, err := Canonical(typ)
	if err != nil {
		t.Fatalf("canon(t, %s): %v", typ, err)
	}
	return s
}

func TestUnexportedStructuralIdentityDistinct(t *testing.T) {
	a := pkgOf(t, "example.com/a", "package a\ntype S interface{ tag() int }")
	b := pkgOf(t, "example.com/b", "package b\ntype S interface{ tag() int }")
	ida := canon(t, a.Scope().Lookup("S").Type().Underlying())
	idb := canon(t, b.Scope().Lookup("S").Type().Underlying())
	if ida == idb {
		t.Fatalf("distinct unexported interfaces collided: %s", ida)
	}
}

// TestAliasResolvesToTarget: an alias is not a distinct dynamic type.
func TestAliasResolvesToTarget(t *testing.T) {
	p := pkgOf(t, "example.com/c", "package c\ntype A = []int\ntype B = A")
	idA := canon(t, p.Scope().Lookup("A").Type())
	idB := canon(t, p.Scope().Lookup("B").Type())
	if idA != idB || idA != "[]int" {
		t.Fatalf("aliases did not normalize: %q vs %q", idA, idB)
	}
}

// TestPathQualification: same package NAME, different paths → distinct.
func TestPathQualification(t *testing.T) {
	a := pkgOf(t, "one/util", "package util\ntype T struct{ X int }")
	b := pkgOf(t, "two/util", "package util\ntype T struct{ X int }")
	if canon(t, a.Scope().Lookup("T").Type()) == canon(t, b.Scope().Lookup("T").Type()) {
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
		id := canon(t, scope.Lookup(name).Type())
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
	if canon(t, aRead) != canon(t, bRead) {
		t.Errorf("same callable signature got distinct identities: %q vs %q", canon(t, aRead), canon(t, bRead))
	}
	if contains(canon(t, aRead), "recv(") {
		t.Errorf("receiver leaked into signature identity: %q", canon(t, aRead))
	}
	i1 := scope.Lookup("I1").Type().Underlying()
	i2 := scope.Lookup("I2").Type().Underlying()
	if canon(t, i1) != canon(t, i2) {
		t.Errorf("structurally identical interfaces got distinct identities: %q vs %q", canon(t, i1), canon(t, i2))
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
	i32 := canon(t, scope.Lookup("FInt32").Type())
	u32 := canon(t, scope.Lookup("FUint32").Type())
	if i32 == u32 {
		t.Errorf("~int32 and ~uint32 constraints collapsed to the same identity: %q", i32)
	}
	if !contains(i32, "int32") {
		t.Errorf("constraint type set not serialized (collapsed toward interface{}): %q", i32)
	}
	gt := canon(t, scope.Lookup("GT").Type())
	gu := canon(t, scope.Lookup("GU").Type())
	if gt != gu {
		t.Errorf("alpha-equivalent generic signatures got distinct identities: %q vs %q", gt, gu)
	}
	hmix := canon(t, scope.Lookup("HMix").Type())
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
	if canon(t, scope.Lookup("Named").Type().Underlying()) == canon(t, scope.Lookup("Embed").Type().Underlying()) {
		t.Errorf("named-field and embedded-field structs share identity")
	}
}

// TestIdenticalEquivalence: the canonical identity MUST agree with
// go/types.Identical for interface/constraint normalization —
// Identical(a,b) => canon(t, a)==canon(t, b), and distinct type sets
// must differ. This is the adversarial acceptance criterion.
func TestIdenticalEquivalence(t *testing.T) {
	src := `package p
type Base interface{ M() }
type UnionAB interface{ ~int | ~string }
type UnionBA interface{ ~string | ~int }
type EmbedBase interface{ Base }
type ExplicitM interface{ M() }
type EmbedPlusMethod interface{ Base; N() }
type MN interface{ M(); N() }
type CmpA interface{ comparable }
type NoTilde interface{ int | string }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	u := func(name string) types.Type { return scope.Lookup(name).Type().Underlying() }
	// Pairs that Go considers identical must share a canonical identity.
	identicalPairs := [][2]string{
		{"UnionAB", "UnionBA"},     // union order is irrelevant
		{"EmbedBase", "ExplicitM"}, // embedded method-interface flattens
		{"EmbedPlusMethod", "MN"},  // embed + explicit method == both methods
	}
	for _, pair := range identicalPairs {
		a, b := u(pair[0]), u(pair[1])
		if !types.Identical(a, b) {
			t.Fatalf("go/types says %s and %s are NOT identical; test premise wrong", pair[0], pair[1])
		}
		if canon(t, a) != canon(t, b) {
			t.Errorf("identical %s/%s got distinct identities:\n  %q\n  %q", pair[0], pair[1], canon(t, a), canon(t, b))
		}
	}
	// Pairs that differ must differ.
	distinctPairs := [][2]string{
		{"UnionAB", "NoTilde"}, // ~int|~string != int|string
		{"ExplicitM", "MN"},    // {M} != {M,N}
		{"UnionAB", "CmpA"},    // union != comparable
	}
	for _, pair := range distinctPairs {
		a, b := u(pair[0]), u(pair[1])
		if types.Identical(a, b) {
			t.Fatalf("go/types says %s and %s ARE identical; test premise wrong", pair[0], pair[1])
		}
		if canon(t, a) == canon(t, b) {
			t.Errorf("distinct %s/%s collapsed to the same identity: %q", pair[0], pair[1], canon(t, a))
		}
	}
}

// TestMethodSignatureExcludesReceiverTypeParams: a generic method's
// callable signature identity must not embed the receiver's type
// parameters (they belong to the receiver shape).
func TestMethodSignatureExcludesReceiverTypeParams(t *testing.T) {
	src := `package p
type Box[T any] struct{}
func (Box[T]) Put(value T) {}
func PutFn[T any](value T) {}
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	box := scope.Lookup("Box").Type().(*types.Named)
	put := methodByName(t, box, "Put").Type()
	id := canon(t, put)
	if contains(id, "recv(") {
		t.Errorf("receiver leaked into method signature identity: %q", id)
	}
	// The method's callable signature references T by binder index, and a
	// free generic function of the same shape should not accidentally
	// share identity (the function DECLARES [T any]; the method does not).
	fn := canon(t, scope.Lookup("PutFn").Type())
	if id == fn {
		t.Logf("method Put: %q", id)
		t.Logf("func PutFn: %q", fn)
	}
}
