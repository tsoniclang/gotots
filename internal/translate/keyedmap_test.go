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
