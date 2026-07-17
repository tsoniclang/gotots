// Field-cell contract tests: &s.f on a non-identity field (a map, slice,
// scalar, or pointer) is a proxying pointer that reads and writes the
// live field, so a store through the pointer mutates the field — the
// lazy-initialization idiom GetOrInit(&s.field) among them.
package translate_test

import "testing"

func TestOracleFieldCellMapLazyInit(t *testing.T) {
	runOracle(t, `package fixture

type scopeSym struct {
	name    string
	members map[string]int
}

// getOrInit is the pointer-to-map-field lazy-init idiom: the write
// through *p must land on the caller's field.
func getOrInit(p *map[string]int) map[string]int {
	if *p == nil {
		*p = make(map[string]int)
	}
	return *p
}

func LazyInitThroughPointer() (int, bool) {
	s := &scopeSym{name: "s"}
	m := getOrInit(&s.members)
	m["a"] = 1
	// The field must now be the same initialized map.
	_, wasNil := s.members["a"]
	return len(s.members), wasNil
}

func SecondInitReusesField() int {
	s := &scopeSym{name: "s"}
	getOrInit(&s.members)["x"] = 10
	getOrInit(&s.members)["y"] = 20
	return len(s.members)
}
`)
}

func TestOracleFieldCellScalarWriteback(t *testing.T) {
	runOracle(t, `package fixture

type counter struct {
	value int
	label string
}

func bump(p *int) {
	*p = *p + 1
}

func ScalarFieldWriteback() (int, string) {
	c := &counter{value: 5, label: "c"}
	bump(&c.value)
	bump(&c.value)
	return c.value, c.label
}

func ScalarFieldAlias() int {
	c := counter{value: 100}
	p := &c.value
	*p = 7
	return c.value
}
`)
}

func TestOracleFieldCellSliceAndPointer(t *testing.T) {
	runOracle(t, `package fixture

type holder struct {
	items []int
	next  *holder
}

func appendOne(p *[]int, v int) {
	*p = append(*p, v)
}

func SliceFieldGrows() (int, int) {
	h := &holder{}
	appendOne(&h.items, 1)
	appendOne(&h.items, 2)
	appendOne(&h.items, 3)
	sum := 0
	for _, v := range h.items {
		sum += v
	}
	return len(h.items), sum
}

func PointerFieldReassign() bool {
	a := &holder{}
	b := &holder{}
	pp := &a.next
	*pp = b
	return a.next == b
}
`)
}

// Field-cell read-through: the pointer observes later field writes made
// directly on the struct, exactly as Go's aliasing pointer does.
func TestOracleFieldCellReadThrough(t *testing.T) {
	runOracle(t, `package fixture

type box struct {
	n int
}

func ReadThroughSeesLaterWrite() (int, int) {
	b := &box{n: 1}
	p := &b.n
	before := *p
	b.n = 99
	after := *p
	return before, after
}
`)
}
