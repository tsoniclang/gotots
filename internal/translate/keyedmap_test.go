// Composite-map-key contract tests: struct keys follow Go value
// equality through the canonical goKey$ encoding — lookups, stores,
// deletes, length, iteration, and pointer-identity fields.
package translate_test

import "testing"

func TestOracleStructMapKeys(t *testing.T) {
	runOracle(t, `package fixture

type cacheKey struct {
	id   int
	name string
	flag bool
}

func ValueEqualityLookup() (int, int, bool) {
	m := map[cacheKey]int{}
	m[cacheKey{id: 1, name: "a", flag: true}] = 10
	m[cacheKey{id: 1, name: "a", flag: true}] = 20
	m[cacheKey{id: 2, name: "a", flag: true}] = 30
	v, ok := m[cacheKey{id: 1, name: "a", flag: true}]
	return v, len(m), ok
}

func MissingKeyZero() (int, bool) {
	m := map[cacheKey]int{}
	v, ok := m[cacheKey{id: 9}]
	return v, ok
}

type twoStrings struct {
	a string
	b string
}

func InjectiveStringFields() int {
	m := map[twoStrings]int{}
	m[twoStrings{a: "x|y", b: "z"}] = 1
	m[twoStrings{a: "x", b: "y|z"}] = 2
	m[twoStrings{a: "x|y|z", b: ""}] = 3
	return len(m)
}

func DeleteAndClear() (int, int) {
	m := map[cacheKey]string{}
	m[cacheKey{id: 1}] = "one"
	m[cacheKey{id: 2}] = "two"
	delete(m, cacheKey{id: 1})
	before := len(m)
	clear(m)
	return before, len(m)
}

func LiteralAndRange() (int, string) {
	m := map[cacheKey]string{
		{id: 1, name: "x"}: "first",
		{id: 2, name: "y"}: "second",
	}
	total := 0
	var got string
	for k, v := range m {
		total += k.id
		if k.name == "y" {
			got = v
		}
	}
	return total, got
}

func KeyMutationAfterInsert() (int, int) {
	k := cacheKey{id: 5}
	m := map[cacheKey]int{}
	m[k] = 50
	k.id = 6
	m[k] = 60
	sum := 0
	for stored := range m {
		sum += stored.id
	}
	return len(m), sum
}

type node struct {
	label string
}

type ptrKey struct {
	target *node
}

func PointerIdentityFields() (int, int, bool) {
	first := &node{label: "a"}
	second := &node{label: "a"}
	m := map[ptrKey]int{}
	m[ptrKey{target: first}] = 1
	m[ptrKey{target: second}] = 2
	m[ptrKey{target: first}] = 3
	v, ok := m[ptrKey{target: first}]
	return len(m), v, ok
}

func NilMapKeyedOps() (int, bool, int) {
	var m map[cacheKey]int
	v, ok := m[cacheKey{id: 1}]
	delete(m, cacheKey{id: 1})
	count := 0
	for range m {
		count++
	}
	return v, ok, count + int(len(m))
}

func NilKeyedMapWritePanics() int {
	var m map[cacheKey]int
	m[cacheKey{id: 1}] = 1
	return 0
}
`)
}

func TestOracleNestedStructMapKeys(t *testing.T) {
	runOracle(t, `package fixture

type span struct {
	pos int32
	end int32
}

type locKey struct {
	file  string
	loc   span
}

func NestedValueEquality() (int, int, bool) {
	m := map[locKey]int{}
	m[locKey{file: "a.go", loc: span{pos: 1, end: 2}}] = 10
	m[locKey{file: "a.go", loc: span{pos: 1, end: 2}}] = 20
	m[locKey{file: "a.go", loc: span{pos: 1, end: 3}}] = 30
	m[locKey{file: "b.go", loc: span{pos: 1, end: 2}}] = 40
	v, ok := m[locKey{file: "a.go", loc: span{pos: 1, end: 2}}]
	return v, len(m), ok
}

func NestedInjectiveBoundary() int {
	// The nested encoding must delimit so that moving a "|" across the
	// nested-struct boundary never collides two distinct keys.
	m := map[locKey]int{}
	m[locKey{file: "x", loc: span{pos: 1, end: 23}}] = 1
	m[locKey{file: "x", loc: span{pos: 12, end: 3}}] = 2
	return len(m)
}

type ptrNode struct {
	label string
}

type nestedPtrKey struct {
	inner  span
	target *ptrNode
}

func NestedPointerIdentity() (int, int, bool) {
	first := &ptrNode{label: "a"}
	second := &ptrNode{label: "a"}
	m := map[nestedPtrKey]int{}
	m[nestedPtrKey{inner: span{pos: 1}, target: first}] = 1
	m[nestedPtrKey{inner: span{pos: 1}, target: second}] = 2
	m[nestedPtrKey{inner: span{pos: 1}, target: first}] = 3
	v, ok := m[nestedPtrKey{inner: span{pos: 1}, target: first}]
	return len(m), v, ok
}

func NestedRangeRecovery() (int32, string) {
	m := map[locKey]string{
		{file: "a", loc: span{pos: 3, end: 4}}: "first",
		{file: "b", loc: span{pos: 5, end: 6}}: "second",
	}
	var total int32
	var got string
	for k, v := range m {
		total += k.loc.pos
		if k.file == "b" {
			got = v
		}
	}
	return total, got
}

type doubleNested struct {
	outer locKey
	rank  int
}

func DoubleNestedEquality() (int, int) {
	m := map[doubleNested]int{}
	m[doubleNested{outer: locKey{file: "a", loc: span{pos: 1, end: 2}}, rank: 7}] = 1
	m[doubleNested{outer: locKey{file: "a", loc: span{pos: 1, end: 2}}, rank: 7}] = 2
	m[doubleNested{outer: locKey{file: "a", loc: span{pos: 1, end: 2}}, rank: 8}] = 3
	sum := 0
	for k := range m {
		sum += k.rank
	}
	return len(m), sum
}
`)
}

func TestOracleTypeParamMapKeys(t *testing.T) {
	runOracle(t, `package fixture

func index[K comparable, V any](keys []K, values []V) map[K]V {
	out := make(map[K]V)
	for i, k := range keys {
		out[k] = values[i]
	}
	return out
}

func TypeParamKeys() (int, string, bool) {
	byInt := index([]int{1, 2}, []string{"one", "two"})
	byString := index([]string{"a"}, []int{10})
	v, ok := byInt[2]
	return byString["a"], v, ok
}
`)
}
