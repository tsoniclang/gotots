package translate_test

import "testing"

func TestOracleSliceSlotIdentity(t *testing.T) {
	// The core.Same shape: &s1[0] == &s2[0] observes backing-store
	// sharing — true for a shared slice and for overlapping subslices,
	// false for equal-valued but distinct backings.
	runOracle(t, `package fixture

func same[T any](s1 []T, s2 []T) bool {
	if len(s1) == len(s2) {
		return len(s1) == 0 || &s1[0] == &s2[0]
	}
	return false
}

func SliceSlotIdentity() int {
	total := 0
	a := []int{1, 2, 3}
	b := a
	c := []int{1, 2, 3}
	if same(a, b) {
		total += 1
	}
	if !same(a, c) {
		total += 10
	}
	if same(a[1:], b[1:]) {
		total += 100
	}
	if !same(a[:2], a[1:]) {
		total += 1000
	}
	var empty1, empty2 []int
	if same(empty1, empty2) {
		total += 10000
	}
	tail := a[2:]
	if &a[2] == &tail[0] {
		total += 100000
	}
	if &a[0] != &tail[0] {
		total += 1000000
	}
	return total
}
`)
}
