package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleSliceOfAnyBoxedIntoAny(t *testing.T) {
	// The tsoptions CompilerOptionsValue shape: a []any value boxed INTO
	// any and asserted back out — the empty-interface union must carry the
	// self-referential "c:[]interface{}" member or the assert compiles to
	// a statically-false panic and everything after it is unreachable.
	runOracle(t, `package fixture

func makeList() any {
	list := []any{"a", 7}
	return list
}

func SliceOfAnyRoundTrip() int {
	v := makeList()
	list, ok := v.([]any)
	if !ok {
		return -1
	}
	total := len(list) * 100
	if s, isStr := list[0].(string); isStr && s == "a" {
		total += 10
	}
	if n, isInt := list[1].(int); isInt && n == 7 {
		total += 1
	}
	return total
}
`)
}

func TestOracleGenericStructEqAndKey(t *testing.T) {
	// The packagejson.Expected shape: a generic struct with a bare-param
	// field, compared with == (delegating to the captured eq$P) and used
	// as a map key at a scalar instantiation (goKeyScalar component).
	runOracle(t, `package fixture

type Expected[T any] struct {
	valid bool
	Value T
}

type Header struct {
	Name    Expected[string]
	Version Expected[string]
}

func GenericStructEqAndKey() int {
	a := Header{Name: Expected[string]{true, "x"}, Version: Expected[string]{true, "1"}}
	b := Header{Name: Expected[string]{true, "x"}, Version: Expected[string]{true, "1"}}
	c := Header{Name: Expected[string]{true, "y"}, Version: Expected[string]{true, "1"}}
	total := 0
	if a == b {
		total += 100
	}
	if a == c {
		total += 10000
	}
	m := map[Header]int{}
	m[a] = 7
	m[c] = 9
	total += m[b] * 10
	total += len(m)
	return total
}
`)
}

func TestOracleGenericMethodPointerIdentity(t *testing.T) {
	// The core.Arena shape: a METHOD on a generic type returning *T where
	// every corpus binding of T is a struct — the receiver's closed
	// instantiation evidence makes *T the identity carrier, so callers
	// store the instance directly into identity-typed locals and fields.
	runOracle(t, `package fixture

type node struct{ v int }

type holder[T any] struct {
	cur *T
}

func (h *holder[T]) Put(p *T) {
	h.cur = p
}

func (h *holder[T]) Get() *T {
	return h.cur
}

func GenericMethodPointerIdentity() int {
	h := &holder[node]{}
	n := &node{v: 40}
	h.Put(n)
	got := h.Get()
	got.v++
	if h.Get() != n {
		return -1
	}
	return n.v
}
`)
}

func TestOracleDoublePointerFieldStore(t *testing.T) {
	// The printer.setTempFlags shape: a field store through an explicit
	// deref of a pointer-to-pointer — (*scope).f keeps BOTH indirections
	// (outer deref in phase one, the store's implicit deref in phase two).
	runOracle(t, `package fixture

type scope struct {
	flags int
}

func set(pp **scope, v int) {
	(*pp).flags = v
}

func DoublePointerFieldStore() int {
	s := &scope{flags: 1}
	pp := &s
	set(pp, 41)
	return s.flags + 1
}
`)
}

func TestOracleClosureAssignedNilSlice(t *testing.T) {
	// The printer comment-collection shape: a nil-initialized slice local
	// assigned ONLY inside a closure, then indexed after the call — the
	// typed nil initializer keeps the declared carrier type so the later
	// element access types exactly.
	runOracle(t, `package fixture

type item struct {
	n int
}

func each(f func(int)) {
	f(1)
	f(2)
}

func ClosureAssignedNilSlice() int {
	var items []item
	each(func(v int) {
		items = append(items, item{n: v * 10})
	})
	if len(items) == 0 {
		return -1
	}
	first := items[0]
	return first.n + items[1].n
}
`)
}

func TestOracleGenericOverGenericZeroField(t *testing.T) {
	// The CopyOnWriteSet shape: a generic struct whose field is ANOTHER
	// generic instantiation over the outer parameter — its zero (inside
	// goZero$) derives eq/clone/set factories for the nested instantiation
	// from the outer scope's own factory parameters.
	runOracle(t, `package fixture

type inner[T any] struct {
	val T
	set bool
}

type outer[T any] struct {
	in inner[T]
}

func (o *outer[T]) Put(v T) {
	o.in = inner[T]{val: v, set: true}
}

func GenericOverGenericZeroField() int {
	var o outer[int]
	if o.in.set {
		return -1
	}
	o.Put(41)
	if !o.in.set {
		return -2
	}
	return o.in.val + 1
}
`)
}

func TestOracleCompoundAssignAndBitOps(t *testing.T) {
	// Registry-coverage oracle: the compound-assignment family (%=, &=,
	// &^=, -=, /=, >>=, ^=), binary |, and non-constant len(array). One
	// compound RHS mutates its own storage to pin Go's load-before-RHS
	// evaluation order.
	runOracle(t, `package fixture

func arr() [4]int {
	return [4]int{1, 2, 3, 4}
}

func CompoundAssignAndBitOps() int {
	x := 0xF3
	x %= 0x51
	x &= 0x7E
	x &^= 0x12
	x -= 3
	x /= 2
	x >>= 1
	x ^= 0x2C
	y := 0x40 | x
	total := y*1000 + len(arr())
	z := 1
	z -= func() int { z = 10; return 2 }()
	return total*100 + z
}
`)
}

func TestOracleCrossPackageGenericMethodIdentity(t *testing.T) {
	// The core.Arena shape ACROSS packages: the generic type lives in one
	// package, every instantiation lives in ANOTHER — the receiver's
	// closed instantiation evidence must span the whole corpus, not the
	// declaring package.
	result, err := oracle.Run(t.TempDir(), map[string]string{
		"holder": `package holder

type Holder[T any] struct {
	cur *T
}

func (h *Holder[T]) Put(p *T) {
	h.cur = p
}

func (h *Holder[T]) Get() *T {
	return h.cur
}
`,
		"fixture": `package fixture

import "oracle.fixture/holder"

type node struct{ v int }

func CrossPackageGenericIdentity() int {
	h := &holder.Holder[node]{}
	n := &node{v: 40}
	h.Put(n)
	got := h.Get()
	got.v++
	if h.Get() != n {
		return -1
	}
	return n.v
}
`,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestOracleGenericStructKeyedSet(t *testing.T) {
	// The collections.Set[NonExistentPropertyKey] shape: a generic map
	// keyed by the type parameter, instantiated with a key-encodable
	// STRUCT — the encoded carrier with the per-binding key$P factory
	// keeps Go's value-key equality exact.
	runOracle(t, `package fixture

type pair struct {
	a int
	b string
}

type set[T comparable] struct {
	m map[T]struct{}
}

func (s *set[T]) Add(v T) {
	if s.m == nil {
		s.m = map[T]struct{}{}
	}
	s.m[v] = struct{}{}
}

func (s *set[T]) Has(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *set[T]) Len() int {
	return len(s.m)
}

func GenericStructKeyedSet() int {
	s := &set[pair]{}
	s.Add(pair{1, "x"})
	s.Add(pair{1, "x"})
	s.Add(pair{2, "y"})
	total := s.Len() * 100
	if s.Has(pair{1, "x"}) {
		total += 10
	}
	if s.Has(pair{3, "z"}) {
		total += 1000
	}
	i := &set[int]{}
	i.Add(7)
	i.Add(7)
	total += i.Len()
	p := &set[*pair]{}
	x := &pair{9, "q"}
	p.Add(x)
	p.Add(x)
	p.Add(&pair{9, "q"})
	total += p.Len() * 10000
	return total
}
`)
}

func TestOracleGenericConstructsKeyCapturingClass(t *testing.T) {
	// The collections.GroupBy shape: a generic function constructs a
	// key-capturing generic class binding its OWN parameters — the hard
	// map-key requirement flows through the instantiation edge to the
	// function's K (guarded bindings), while V stays soft-free.
	runOracle(t, `package fixture

type multi[K comparable, V any] struct {
	m map[K][]V
}

func (s *multi[K, V]) Add(key K, value V) {
	if s.m == nil {
		s.m = map[K][]V{}
	}
	s.m[key] = append(s.m[key], value)
}

func (s *multi[K, V]) Count(key K) int {
	return len(s.m[key])
}

type pair struct {
	a int
	b string
}

func groupBy[K comparable, V any](items []V, id func(V) K) *multi[K, V] {
	m := &multi[K, V]{}
	for _, item := range items {
		m.Add(id(item), item)
	}
	return m
}

func GenericConstructsKeyCapturingClass() int {
	byLen := groupBy([]string{"a", "bb", "cc", "ddd"}, func(s string) int { return len(s) })
	total := byLen.Count(2) * 100
	byKey := groupBy([]int{1, 2, 3, 4, 5}, func(n int) pair { return pair{a: n % 2, b: "x"} })
	total += byKey.Count(pair{a: 1, b: "x"}) * 10
	total += byKey.Count(pair{a: 0, b: "x"})
	return total
}
`)
}

func TestOracleInstantiatedGenericInterfaceValue(t *testing.T) {
	// The packagejson.Expected/TypeValidatedField shape: pointers to
	// CONCRETE INSTANTIATIONS of a generic struct boxed into a non-empty
	// interface — each instantiation is a composite-branded union member
	// with an inline vtable over the generated generic functions.
	runOracle(t, `package fixture

type box[T any] struct {
	val   T
	valid bool
}

func (b *box[T]) IsValid() bool {
	return b.valid
}

func (b *box[T]) Weight() int {
	if b.valid {
		return 2
	}
	return 1
}

type checker interface {
	IsValid() bool
	Weight() int
}

func InstantiatedGenericInterfaceValue() int {
	a := &box[string]{val: "x", valid: true}
	b := &box[int]{val: 7, valid: false}
	var checks []checker
	checks = append(checks, a, b)
	total := 0
	for _, c := range checks {
		if c.IsValid() {
			total += 100
		}
		total += c.Weight()
	}
	if sb, ok := checks[0].(*box[string]); ok && sb.val == "x" {
		total += 1000
	}
	if _, ok := checks[1].(*box[string]); ok {
		total += 100000
	}
	return total
}
`)
}

func TestOraclePointerCarrierAdmissions(t *testing.T) {
	// Three admissions in one differential: pointer to a named ARRAY type
	// (identity), pointer to a named scalar carrier through a cell, and
	// the same-underlying owned pointer conversion (type Mutable Base).
	runOracle(t, `package fixture

type table [2]int

type counter int

type base struct {
	n int
}

type mutable base

func bump(c *counter) {
	*c += 5
}

func PointerCarrierAdmissions() int {
	var t table
	pt := &t
	pt[0] = 3
	pt[1] = 4
	var c counter = 10
	bump(&c)
	m := &mutable{n: 20}
	b := (*base)(m)
	b.n++
	return t[0]*100 + t[1]*10 + int(c) + m.n*1000
}
`)
}

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
