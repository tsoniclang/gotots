// Three-index slice contract tests: s[low:high:max] shares the operand's
// backing but caps the result at max-low, so a later append reallocates
// once the limited capacity is exhausted instead of overwriting the
// operand's tail — Go's capacity-limiting full slice expression.
package translate_test

import "testing"

func TestOracleThreeIndexSliceCapacity(t *testing.T) {
	runOracle(t, `package fixture

func CapacityLimited() (int, int, int) {
	s := make([]int, 5, 10)
	t := s[1:3:4]
	return len(t), cap(t), len(s)
}

// The three-index cap forces append to reallocate, so the operand's tail
// is not overwritten (the classic full-slice safety idiom).
func AppendReallocatesNoAlias() (int, int) {
	backing := []int{1, 2, 3, 4, 5}
	head := backing[0:2:2]
	head = append(head, 99) // cap exhausted -> reallocates
	// backing[2] must be untouched (still 3), not overwritten by 99.
	return backing[2], head[2]
}

// Without the cap limit, append into spare capacity aliases the backing.
func TwoIndexAppendAliases() (int, int) {
	backing := []int{1, 2, 3, 4, 5}
	head := backing[0:2] // cap is 5, spare capacity available
	head = append(head, 99)
	return backing[2], head[2]
}

func OmittedLowBound() (int, int) {
	s := make([]int, 6, 12)
	t := s[:2:3]
	return len(t), cap(t)
}
`)
}

func TestOracleThreeIndexSliceBounds(t *testing.T) {
	runOracle(t, `package fixture

// Each case lets the bad slice panic uncaught; the differential driver
// captures the exact panic message on both sides, so the three-index
// bound messages match Go byte for byte. Variable indices force the
// check to run time rather than a Go compile error.

// max > cap panics: [::max] with capacity cap.
func MaxExceedsCapPanics() int {
	s := make([]int, 3, 5)
	lo, hi, mx := 0, 2, 9
	x := s[lo:hi:mx]
	return len(x)
}

// high > max panics: [:high:max].
func HighExceedsMaxPanics() int {
	s := make([]int, 5, 8)
	lo, hi, mx := 0, 4, 3
	x := s[lo:hi:mx]
	return len(x)
}

// low > high panics: [low:high:].
func LowExceedsHighPanics() int {
	s := make([]int, 5, 8)
	lo, hi, mx := 3, 2, 6
	x := s[lo:hi:mx]
	return len(x)
}
`)
}
