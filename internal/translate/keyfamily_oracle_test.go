// Key-family monomorphization and pointer/conversion admission oracles.
package translate_test

import "testing"

func TestOracleMixedFamilyGenericSet(t *testing.T) {
	// The collections.Set shape with MIXED key families: scalar
	// instantiations keep the direct SVZ carrier (their exported maps
	// cross the generic boundary representation-identically) while a
	// struct instantiation takes the "$ek" encoded-family variant — two
	// exact emissions, one Go type per family everywhere.
	runOracle(t, `package fixture

type set[T comparable] struct {
	m map[T]struct{}
}

func (s *set[T]) Add(v T) {
	if s.m == nil {
		s.m = map[T]struct{}{}
	}
	s.m[v] = struct{}{}
}

func (s *set[T]) Keys() map[T]struct{} {
	out := map[T]struct{}{}
	for k := range s.m {
		out[k] = struct{}{}
	}
	return out
}

type pt struct {
	x int
	y int
}

func MixedFamilyGenericSet() int {
	ss := &set[string]{}
	ss.Add("a")
	ss.Add("b")
	ss.Add("a")
	exported := ss.Keys()
	if _, ok := exported["b"]; !ok {
		return -1
	}
	direct := map[string]struct{}{"c": {}}
	for k := range exported {
		direct[k] = struct{}{}
	}
	sp := &set[pt]{}
	sp.Add(pt{1, 2})
	sp.Add(pt{1, 2})
	sp.Add(pt{3, 4})
	total := len(direct)*100 + len(sp.Keys())*10 + len(exported)
	return total
}
`)
}

func TestOracleGenericFuncValueReference(t *testing.T) {
	// The core.Identity-as-callback shape: an implicitly instantiated
	// generic function referenced as a first-class value eta-expands to
	// an exactly typed arrow over the instantiation's derivations.
	runOracle(t, `package fixture

func identity[T any](t T) T {
	return t
}

func apply(f func(int) int, v int) int {
	return f(v)
}

func GenericFuncValueReference() int {
	f := identity[int]
	total := apply(f, 20)
	total += apply(identity[int], 3)
	return total
}
`)
}

func TestOracleLocalTypeDeclaration(t *testing.T) {
	// The checker GetResolvedSignatureForSignatureHelp shape: a LOCAL
	// struct type declared inside a body — its class synthesizes at
	// module level through the anonymous-struct pipeline; two same-named
	// locals in different functions stay distinct.
	runOracle(t, `package fixture

func first() int {
	type result struct {
		a int
		b int
	}
	r := result{a: 3, b: 4}
	return r.a*10 + r.b
}

func second() int {
	type result struct {
		x string
	}
	r := result{x: "abcd"}
	return len(r.x)
}

func LocalTypeDeclaration() int {
	return first()*100 + second()
}
`)
}

func TestOracleAnyKeyedGenericMap(t *testing.T) {
	// The tsc/help NewOrderedMapWithSizeHint[any, []string] shape: an
	// INTERFACE binding of a hard map-keyed parameter joins the encoded
	// family — the union $key encoder is the instantiation's key$P.
	runOracle(t, `package fixture

type dict[K comparable, V any] struct {
	m map[K]V
}

func (d *dict[K, V]) Put(k K, v V) {
	if d.m == nil {
		d.m = map[K]V{}
	}
	d.m[k] = v
}

func (d *dict[K, V]) Get(k K) V {
	return d.m[k]
}

func (d *dict[K, V]) Len() int {
	return len(d.m)
}

func AnyKeyedGenericMap() int {
	d := &dict[any, int]{}
	d.Put("a", 1)
	d.Put(7, 2)
	d.Put("a", 3)
	total := d.Len()*100 + d.Get("a")*10 + d.Get(7)
	s := &dict[string, int]{}
	s.Put("x", 9)
	return total*10 + s.Get("x")
}
`)
}

func TestOracleAddressOfTupleBound(t *testing.T) {
	// The tracing shape: a multi-result binding whose name is addressed —
	// the stable cell declares off the tuple slot, aliasing exactly.
	runOracle(t, `package fixture

func two() (int, int) {
	return 3, 4
}

func bump(p *int) {
	*p += 10
}

func AddressOfTupleBound() int {
	a, b := two()
	bump(&b)
	return a*100 + b
}
`)
}

func TestOracleParamZeroDeref(t *testing.T) {
	// The core.ElementOrNil shape: *new(T) is Go's zero-of-parameter
	// idiom — each evaluation a fresh zero of the binding.
	runOracle(t, `package fixture

type rec struct {
	n int
}

func elementOrNil[T any](slice []T, index int) T {
	if index < len(slice) {
		return slice[index]
	}
	return *new(T)
}

func ParamZeroDeref() int {
	nums := []int{7, 8}
	total := elementOrNil(nums, 1)*10 + elementOrNil(nums, 5)
	r := elementOrNil([]rec{{n: 3}}, 9)
	s := elementOrNil([]string{"x"}, 2)
	return total*100 + r.n*10 + len(s)
}
`)
}

func TestOracleClearSliceAndNamedConversions(t *testing.T) {
	// clear(s) zeroes elements IN PLACE (aliases observe it); named
	// same-underlying conversions (map and struct forms) are identity.
	runOracle(t, `package fixture

type scores map[string]int

type point struct {
	x int
}

type spot point

func ClearSliceAndNamedConversions() int {
	s := []int{5, 6, 7}
	alias := s
	clear(s)
	m := scores{"a": 1}
	plain := map[string]int(m)
	plain["b"] = 2
	back := scores(plain)
	p := spot(point{x: 9})
	q := point(p)
	return alias[0]*1000 + len(back)*100 + p.x*10 + q.x
}
`)
}

func TestOracleCopyStringAndUnitBox(t *testing.T) {
	// copy([]byte, string) copies the exact UTF-8 bytes; struct{}{} boxes
	// as one interned composite member of the empty interface.
	runOracle(t, `package fixture

func CopyStringAndUnitBox() int {
	buf := make([]byte, 5)
	n := copy(buf, "héllo")
	var v any = struct{}{}
	unit := 0
	if _, ok := v.(struct{}); ok {
		unit = 1
	}
	return n*100 + int(buf[0])/10 + unit
}
`)
}

func TestOracleCoreTypedParams(t *testing.T) {
	// The core.BinarySearchUniqueFunc shape: S ~[]E erases to its slice
	// carrier — len/index/range work directly; named-slice bindings share
	// the carrier identically.
	runOracle(t, `package fixture

type ints []int

func at[S ~[]E, E any](x S, i int) E {
	return x[i]
}

func total[S ~[]E, E any](x S, f func(E) int) int {
	n := 0
	for i := 0; i < len(x); i++ {
		n += f(at(x, i))
	}
	return n
}

func CoreTypedParams() int {
	plain := []string{"a", "bb"}
	named := ints{3, 4, 5}
	a := total(plain, func(s string) int { return len(s) })
	b := total(named, func(v int) int { return v })
	return a*100 + b
}
`)
}

func toUint[T ~uint32](v T) uint32 {
	return uint32(v)
}
