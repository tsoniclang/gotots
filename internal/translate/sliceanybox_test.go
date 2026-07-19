package translate_test

import "testing"

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
