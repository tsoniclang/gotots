package typeset

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// typecheck parses and type-checks one file, returning the package, Info, and
// any type error. The toolchain's own checker is the differential oracle.
func typecheck(t *testing.T, src string) (*types.Package, *types.Info, error) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "m.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info := &types.Info{
		Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{},
		Uses: map[*ast.Ident]types.Object{},
	}
	config := &types.Config{Importer: importer.Default(), Error: func(error) {}}
	pkg, err := config.Check("m", fset, []*ast.File{file}, info)
	return pkg, info, err
}

// typeParamOf finds the first type parameter of the named generic function.
func typeParamOf(t *testing.T, pkg *types.Package, fn string) *types.TypeParam {
	t.Helper()
	obj, ok := pkg.Scope().Lookup(fn).(*types.Func)
	if !ok {
		t.Fatalf("no function %s", fn)
	}
	params := obj.Type().(*types.Signature).TypeParams()
	if params.Len() == 0 {
		t.Fatalf("%s has no type parameters", fn)
	}
	return params.At(0)
}

// varType finds a declared variable's type inside the checked Info.
func varType(t *testing.T, info *types.Info, name string) types.Type {
	t.Helper()
	for ident, obj := range info.Defs {
		if ident.Name == name && obj != nil {
			if v, ok := obj.(*types.Var); ok {
				return v.Type()
			}
		}
	}
	t.Fatalf("no variable %s", name)
	return nil
}

// TestCoreAgreesWithRangeOperation proves Core against a toolchain-typed
// OPERATION: the range value variable's type is the toolchain's statement of
// the constraint's core element type.
func TestCoreAgreesWithRangeOperation(t *testing.T) {
	pkg, info, err := typecheck(t, `package m

type Ints []int
type Named []int

func Range[S ~[]int](s S) {
	for _, v := range s {
		_ = v
	}
}

func RangeNamed[S Ints | Named | []int](s S) {
	for _, w := range s {
		_ = w
	}
}
`)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	for fn, varName := range map[string]string{"Range": "v", "RangeNamed": "w"} {
		core, ok := Core(typeParamOf(t, pkg, fn))
		if !ok {
			t.Fatalf("%s: no core type", fn)
		}
		slice, ok := core.(*types.Slice)
		if !ok {
			t.Fatalf("%s: core = %v, want slice", fn, core)
		}
		if declared := varType(t, info, varName); !types.Identical(slice.Elem(), declared) {
			t.Errorf("%s: core element %v diverges from toolchain-typed range variable %v", fn, slice.Elem(), declared)
		}
	}
}

// TestCoreAgreesWithReceiveOperation proves the directional-channel merge
// against toolchain-accepted receive and toolchain-rejected send.
func TestCoreAgreesWithReceiveOperation(t *testing.T) {
	pkg, info, err := typecheck(t, `package m

func Recv[C chan int | <-chan int](c C) {
	r := <-c
	_ = r
}
`)
	if err != nil {
		t.Fatalf("receive over merged channels must typecheck: %v", err)
	}
	core, ok := Core(typeParamOf(t, pkg, "Recv"))
	if !ok {
		t.Fatal("no core for merged channel constraint")
	}
	channel, ok := core.(*types.Chan)
	if !ok || channel.Dir() != types.RecvOnly {
		t.Fatalf("core = %v, want <-chan int (most restrictive direction)", core)
	}
	if declared := varType(t, info, "r"); !types.Identical(channel.Elem(), declared) {
		t.Errorf("core element %v diverges from toolchain-typed receive %v", channel.Elem(), declared)
	}
	// The toolchain rejects SEND through the same constraint — and Core's
	// direction predicts exactly that.
	if _, _, err := typecheck(t, `package m

func Send[C chan int | <-chan int](c C) {
	c <- 1
}
`); err == nil {
		t.Error("toolchain accepted send through a recv-only core; differential broken")
	}
	// Conflicting directions have no core; the toolchain rejects receive too.
	if _, _, err := typecheck(t, `package m

func Conflict[C chan<- int | <-chan int](c C) {
	<-c
}
`); err == nil {
		t.Error("toolchain accepted receive over direction-conflicting channels")
	}
	conflictSrc := `package m

func Conflict[C chan<- int | <-chan int](c C) {}
`
	pkgConflict, _, err := typecheck(t, conflictSrc)
	if err != nil {
		t.Fatalf("declaration-only conflicting constraint is legal: %v", err)
	}
	if _, ok := Core(typeParamOf(t, pkgConflict, "Conflict")); ok {
		t.Error("Core produced a type for direction-conflicting channels")
	}
}

// TestMixedAndEmptySetsHaveNoCore proves legal-but-coreless declarations:
// the declaration typechecks, the operation is toolchain-rejected, and Core
// says false.
func TestMixedAndEmptySetsHaveNoCore(t *testing.T) {
	declSrc := `package m

func Mixed[T int | chan int](x T) {}

func Unconstrained[T any](x T) {}
`
	pkg, _, err := typecheck(t, declSrc)
	if err != nil {
		t.Fatalf("legal declarations rejected: %v", err)
	}
	if _, ok := Core(typeParamOf(t, pkg, "Mixed")); ok {
		t.Error("mixed type set produced a core type")
	}
	if _, ok := Core(typeParamOf(t, pkg, "Unconstrained")); ok {
		t.Error("unconstrained any produced a core type")
	}
	// The operation-level proof: range over the mixed param is rejected by
	// the toolchain, exactly as no-core predicts.
	if _, _, err := typecheck(t, `package m

func Mixed[T int | chan int](x T) {
	for range x {
	}
}
`); err == nil {
		t.Error("toolchain accepted range over a coreless type set")
	}
}

// TestTildeTermsAndIntersections proves tilde underlying contribution and
// embedded-interface intersection through a toolchain-typed operation.
func TestTildeTermsAndIntersections(t *testing.T) {
	pkg, info, err := typecheck(t, `package m

type MyString string

type Stringish interface{ ~string }

func Index[S Stringish](s S) {
	b := s[0]
	_ = b
}
`)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	core, ok := Core(typeParamOf(t, pkg, "Index"))
	if !ok {
		t.Fatal("tilde constraint has no core")
	}
	basic, ok := core.(*types.Basic)
	if !ok || basic.Kind() != types.String {
		t.Fatalf("core = %v, want string", core)
	}
	if declared := varType(t, info, "b"); (declared.(*types.Basic)).Kind() != types.Byte {
		t.Errorf("index result %v diverges from string-core prediction (byte)", declared)
	}
}

// TestFoilApproximationDisagrees embeds the forbidden construct-local
// flatten/deduplicate approximation (no channel-direction rule, no tilde
// underlying) and proves the differential corpus distinguishes it from the
// owner — restoring such an approximation in production fails this matrix.
func TestFoilApproximationDisagrees(t *testing.T) {
	foil := func(tp *types.TypeParam) (types.Type, bool) {
		iface, ok := types.Unalias(tp.Constraint()).Underlying().(*types.Interface)
		if !ok || iface.NumEmbeddeds() == 0 {
			return nil, false
		}
		var first types.Type
		same := true
		for i := 0; i < iface.NumEmbeddeds(); i++ {
			if union, ok := types.Unalias(iface.EmbeddedType(i)).(*types.Union); ok {
				for j := 0; j < union.Len(); j++ {
					u := types.Unalias(union.Term(j).Type()).Underlying()
					if first == nil {
						first = u
					} else if !types.Identical(first, u) {
						same = false
					}
				}
			}
		}
		return first, same && first != nil
	}
	pkg, _, err := typecheck(t, `package m

func Merged[C chan int | <-chan int](c C) {}
`)
	if err != nil {
		t.Fatal(err)
	}
	tp := typeParamOf(t, pkg, "Merged")
	ownerCore, ownerOK := Core(tp)
	_, foilOK := foil(tp)
	if foilOK == ownerOK {
		t.Fatalf("foil approximation agrees with the owner on the channel corpus (owner ok=%v core=%v); the matrix does not distinguish them", ownerOK, ownerCore)
	}
}
