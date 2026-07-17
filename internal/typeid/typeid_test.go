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

// TestTotalContract: Canonical is a total (string,error) contract with no
// in-band poison marker — an ordinary type returns a clean identity, and
// a type with no exact identity (a free type parameter with no binder)
// returns an ERROR directly.
func TestTotalContract(t *testing.T) {
	src := `package p
type T struct{ x int }
func F[A any](a A) {}
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	if _, err := Canonical(scope.Lookup("T").Type()); err != nil {
		t.Errorf("ordinary type returned an error: %v", err)
	}
	// A[A].M's callable references A (free receiver-like param); a bare
	// Canonical of a free parameter fails closed.
	fsig := scope.Lookup("F").Type().(*types.Signature)
	free := fsig.Params().At(0).Type() // the type parameter A, free here
	if _, err := Canonical(free); err == nil {
		t.Errorf("a free type parameter should fail closed, got no error")
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
	put := methodByName(t, box, "Put")
	id, err := MethodCanonical(put)
	if err != nil {
		t.Fatalf("MethodCanonical(Put): %v", err)
	}
	if contains(id, "recv(") {
		t.Errorf("receiver leaked into method signature identity: %q", id)
	}
	// A standalone Canonical of a method signature with free receiver
	// parameters must FAIL closed — there is no global binder.
	if _, err := Canonical(put.Type()); err == nil {
		t.Errorf("Canonical of a method signature with free receiver params should fail closed")
	}
	// A free generic function's own type parameters ARE part of its
	// callable identity (bound and constrained), so PutFn is well-formed.
	if _, err := canonErr(scope.Lookup("PutFn").Type()); err != nil {
		t.Errorf("free generic function identity failed: %v", err)
	}
}

func canonErr(t types.Type) (string, error) { return Canonical(t) }

// TestTypeSetAlgebra checks the reviewer's exact counterexamples: the
// canonical identity must agree with go/types.Identical for constraint
// type-set intersection, empty sets, and comparable filtering.
func TestTypeSetAlgebra(t *testing.T) {
	src := `package p
type UnionIS   interface{ int | string }
type IntersIS  interface{ int; string }
type UnionThenInt interface{ int | string; int }
type JustInt   interface{ int }
type CmpInt    interface{ comparable; int }
type EmptyA    interface{ int; string }
type EmptyB    interface{ bool; float64 }
type CmpTop    interface{ comparable }
type EmptyIface interface{}
type Tilde32   interface{ ~int32 }
type TildeU32  interface{ ~uint32 }
type MethodM   interface{ M() }
type EmbedM    interface{ MethodM }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	u := func(name string) types.Type { return scope.Lookup(name).Type().Underlying() }
	// (a, b, mustBeIdentical) — verified against go/types.Identical and asserted on Canonical.
	cases := []struct {
		a, b string
	}{
		{"UnionThenInt", "JustInt"}, // {int}∩{int,string}={int}
		{"CmpInt", "JustInt"},       // comparable;int == int (int is comparable)
		{"EmptyA", "EmptyB"},        // ∅ == ∅
		{"EmbedM", "MethodM"},       // embedded method interface flattens
	}
	for _, c := range cases {
		a, b := u(c.a), u(c.b)
		if !types.Identical(a, b) {
			t.Fatalf("premise: go/types says %s and %s differ", c.a, c.b)
		}
		if canon(t, a) != canon(t, b) {
			t.Errorf("identical %s/%s got distinct identities:\n  %q\n  %q", c.a, c.b, canon(t, a), canon(t, b))
		}
	}
	distinct := []struct{ a, b string }{
		{"UnionIS", "IntersIS"},  // {int,string} != ∅
		{"CmpTop", "EmptyIface"}, // comparable != interface{}
		{"Tilde32", "TildeU32"},  // ~int32 != ~uint32
		{"UnionIS", "JustInt"},   // {int,string} != {int}
	}
	for _, c := range distinct {
		a, b := u(c.a), u(c.b)
		if types.Identical(a, b) {
			t.Fatalf("premise: go/types says %s and %s are identical", c.a, c.b)
		}
		if canon(t, a) == canon(t, b) {
			t.Errorf("distinct %s/%s collapsed to identity: %q", c.a, c.b, canon(t, a))
		}
	}
}

// TestTupleArity: a one-element tuple must not collide with its element.
func TestTupleArity(t *testing.T) {
	src := `package p
func One() int { return 0 }
func Two() (int, int) { return 0, 0 }
`
	pkg := pkgOf(t, "p", src)
	scope := pkg.Scope()
	one := scope.Lookup("One").Type().(*types.Signature).Results()
	elem := scope.Lookup("One").Type().(*types.Signature).Results().At(0).Type()
	if canon(t, one) == canon(t, elem) {
		t.Errorf("1-tuple serializes like its element: %q", canon(t, one))
	}
}

// TestUnexportedFieldStructsPackageDistinct: two anonymous struct{x int}
// with UNEXPORTED fields from different packages are distinct Go types
// and must not share an identity; exported-field shapes coincide.
func TestUnexportedFieldStructsPackageDistinct(t *testing.T) {
	a := pkgOf(t, "a", "package a\ntype T struct{ x int }\n")
	b := pkgOf(t, "b", "package b\ntype T struct{ x int }\n")
	e1 := pkgOf(t, "c", "package c\ntype T struct{ X int }\n")
	e2 := pkgOf(t, "d", "package d\ntype T struct{ X int }\n")
	under := func(p *types.Package) types.Type { return p.Scope().Lookup("T").Type().Underlying() }
	if canon(t, under(a)) == canon(t, under(b)) {
		t.Errorf("unexported-field structs from different packages share identity: %q", canon(t, under(a)))
	}
	if canon(t, under(e1)) != canon(t, under(e2)) {
		t.Errorf("exported-field structs from different packages should coincide: %q vs %q", canon(t, under(e1)), canon(t, under(e2)))
	}
}
