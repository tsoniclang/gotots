// Generics value-copy differential proof (plan A1): struct and array type
// arguments instantiate with EXACT Go copy semantics through the factory
// protocol (clone$P at bind sites, set$P at storage sites).
package translate_test

import "testing"

func TestOracleGenericStructArg(t *testing.T) {
	// A struct binding: the generic body's assignment copies; mutating one
	// result leaves the other and the source unchanged.
	runOracle(t, `package fixture

type pt struct{ X, Y int }

func dup[T any](v T) (T, T) {
	x := v
	return x, v
}

func GenericStructArg() int {
	a := pt{X: 1, Y: 2}
	b, c := dup(a)
	b.X = 100
	return a.X*10000 + b.X*10 + c.X
}
`)
}

func TestOracleGenericStructReturnBoundary(t *testing.T) {
	// The returned T-bound struct is independent of callee-held state.
	runOracle(t, `package fixture

type box struct{ N int }

func identity[T any](v T) T { return v }

func GenericStructReturn() int {
	src := box{N: 7}
	got := identity(src)
	got.N = 50
	return src.N*100 + got.N
}
`)
}

func TestOracleGenericArrayArg(t *testing.T) {
	// A fixed-array binding copies element-wise.
	runOracle(t, `package fixture

func first[T any](v T) T {
	w := v
	return w
}

func GenericArrayArg() int {
	a := [2]int{3, 4}
	b := first(a)
	b[0] = 99
	return a[0]*100 + b[0]
}
`)
}

func TestOracleGenericSliceOfStructElem(t *testing.T) {
	// []T with a struct binding: element stores in the generic body are
	// observed by concrete aliases of the same backing (set$P in place).
	runOracle(t, `package fixture

type cell struct{ V int }

func setFirst[T any](s []T, v T) {
	s[0] = v
}

func GenericSliceElem() int {
	s := []cell{{V: 1}, {V: 2}}
	alias := s[0:1]
	setFirst(s, cell{V: 42})
	return alias[0].V*10 + s[1].V
}
`)
}

func TestOracleGenericZeroFreshness(t *testing.T) {
	// zero$P yields distinct fresh instances for a struct binding.
	runOracle(t, `package fixture

type z struct{ N int }

func zeros[T any](n int) []T {
	return make([]T, n)
}

func GenericZeroFreshness() int {
	s := zeros[z](2)
	s[0].N = 5
	return s[0].N*10 + s[1].N
}
`)
}

func TestOracleGenericEqStructNaN(t *testing.T) {
	// eq$P for a comparable struct binding uses goEq$; a float field keeps
	// NaN self-inequality exact.
	runOracle(t, `package fixture

type fpt struct{ F float64 }

func contains[T comparable](s []T, v T) bool {
	for _, e := range s {
		if e == v {
			return true
		}
	}
	return false
}

func GenericEqStruct() int {
	a := fpt{F: 1.5}
	n := 0
	if contains([]fpt{{F: 1.5}}, a) {
		n += 1
	}
	nan := fpt{F: nan()}
	if contains([]fpt{nan}, nan) {
		n += 10
	}
	return n
}

func nan() float64 {
	z := 0.0
	return z / z
}
`)
}

func TestOracleGenericClassStructBinding(t *testing.T) {
	// A generic TYPE instantiated with a struct: the class captures the
	// binding's clone/set factories — whole-value copy of the generic
	// struct keeps field instances independent; whole-value overwrite is
	// observed through an alias.
	runOracle(t, `package fixture

type inner struct{ N int }

type holder[T any] struct {
	V T
}

func GenericClassStruct() int {
	a := holder[inner]{V: inner{N: 1}}
	b := a
	b.V.N = 50
	p := &a
	p.V.N = 7
	return a.V.N*100 + b.V.N
}
`)
}

func TestOracleGenericClassZero(t *testing.T) {
	// The zero of a struct-instantiated generic class is fresh per field.
	runOracle(t, `package fixture

type leaf struct{ N int }

type wrap[T any] struct {
	A T
	B T
}

func GenericClassZero() int {
	var w wrap[leaf]
	w.A.N = 9
	return w.A.N*10 + w.B.N
}
`)
}
