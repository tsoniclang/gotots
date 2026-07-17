// Field-address contract tests: &s.f on a non-identity field is the
// field's one stable per-instance cell, so repeated &s.f is the same
// pointer, the address keys a map by Go pointer identity, and a store
// through the pointer and a direct field write are mutually visible.
// Only address-taken fields carry the cell representation.
package translate_test

import "testing"

// Repeated field-address equality and pointer-identity map keys.
func TestOracleFieldAddressIdentity(t *testing.T) {
	runOracle(t, `package fixture

type box struct {
	n int
}

func RepeatedAddressEqual() bool {
	value := &box{}
	return &value.n == &value.n
}

func DistinctInstancesDistinctAddresses() bool {
	a := &box{}
	b := &box{}
	return &a.n == &b.n
}

func FieldAddressMapKey() int {
	value := &box{}
	table := map[*int]int{}
	table[&value.n] = 1
	table[&value.n] = 2
	return len(table)
}

func TwoFieldsDistinctKeys() int {
	value := &twoScalars{}
	table := map[*int]int{}
	table[&value.a] = 1
	table[&value.b] = 2
	return len(table)
}

type twoScalars struct {
	a int
	b int
}
`)
}

// A store through a captured pointer and a direct field write are
// mutually visible; the cell keeps one storage location.
func TestOracleFieldAddressAliasing(t *testing.T) {
	runOracle(t, `package fixture

type cell struct {
	n int
}

func WriteThroughPointerSeenDirectly() (int, int) {
	c := &cell{}
	p := &c.n
	*p = 42
	before := c.n
	c.n = 7
	return before, *p
}

func DirectWriteSeenThroughPointer() (int, int) {
	c := &cell{n: 1}
	p := &c.n
	c.n = 99
	return *p, c.n
}

// The pointer aliases the field across a helper call boundary.
func bump(p *int) {
	*p = *p + 1
}

func AliasAcrossParameter() int {
	c := &cell{n: 10}
	bump(&c.n)
	bump(&c.n)
	return c.n
}
`)
}

// Cell fields of every non-identity kind: pointer, map, slice, and the
// pointer-to-map-field lazy-init idiom.
func TestOracleFieldAddressKinds(t *testing.T) {
	runOracle(t, `package fixture

type holder struct {
	ptr   *int
	table map[string]int
	items []int
}

func initMap(p *map[string]int) {
	if *p == nil {
		*p = make(map[string]int)
	}
}

func LazyInitMapField() (int, bool) {
	h := &holder{}
	initMap(&h.table)
	h.table["a"] = 1
	_, ok := h.table["a"]
	return len(h.table), ok
}

func appendField(p *[]int, v int) {
	*p = append(*p, v)
}

func SliceFieldGrows() int {
	h := &holder{}
	appendField(&h.items, 1)
	appendField(&h.items, 2)
	return len(h.items)
}

func setPtr(pp **int, v *int) {
	*pp = v
}

func PointerFieldReassign() int {
	h := &holder{}
	n := 5
	setPtr(&h.ptr, &n)
	return *h.ptr
}
`)
}

// A whole-struct value copy gives the copy its own distinct field
// storage: &copy.f is not &original.f, and mutating one is independent.
func TestOracleFieldAddressCopyDistinct(t *testing.T) {
	runOracle(t, `package fixture

type point struct {
	x int
}

func addr(p *point) *int {
	return &p.x
}

func CopyHasDistinctFieldStorage() (bool, int, int) {
	original := point{x: 1}
	copy := original // value copy
	copy.x = 9
	// &original.x and &copy.x must be different locations.
	same := addr(&original) == addr(&copy)
	return same, original.x, copy.x
}
`)
}

// An embedded struct's address-taken field carries the cell, and &s.f
// through promotion is exact.
func TestOracleFieldAddressEmbedded(t *testing.T) {
	runOracle(t, `package fixture

type inner struct {
	n int
}

type outer struct {
	inner
	tag int
}

func EmbeddedFieldAddress() bool {
	o := &outer{}
	return &o.n == &o.n
}

func EmbeddedWriteThroughPointer() int {
	o := &outer{}
	p := &o.n
	*p = 33
	return o.n
}
`)
}

// A generic struct's field address must be keyed by the struct's origin,
// so distinct instances have distinct field addresses (the address scan
// sees an instantiated field object while class emission sees the generic
// declaration — they must agree on the storage identity).
func TestOracleGenericFieldAddress(t *testing.T) {
	runOracle(t, `package fixture

type Box[T any] struct {
	X T
}

func DistinctGenericInstances() (bool, bool) {
	a := &Box[int]{X: 5}
	b := &Box[int]{X: 5}
	return &a.X == &b.X, &a.X == &a.X
}

func GenericWriteThroughPointer() int {
	box := &Box[int]{}
	p := &box.X
	*p = 42
	return box.X
}

func TwoDistinctInstantiations() bool {
	a := &Box[int]{X: 1}
	b := &Box[string]{X: "x"}
	_ = b
	p := &a.X
	*p = 9
	return a.X == 9
}
`)
}
