// Multi-result-into-variadic contract tests: Go's f(g()) where f ends in
// a variadic parameter binds g's leading results to the regular
// parameters and packs the rest into the final slice, evaluating g once.
// The remaining-results-empty case leaves the variadic the nil slice,
// exactly as a call with no variadic arguments; a pure variadic target
// (no fixed parameters) is legal and packs every result.
package translate_test

import "testing"

func TestOracleTupleVariadicForwarding(t *testing.T) {
	runOracle(t, `package fixture

func sum(scale int, rest ...int) int {
	total := 0
	for _, v := range rest {
		total += v
	}
	return total * scale
}

func threeInts() (int, int, int) {
	return 2, 3, 4
}

// f(g()): scale binds the first result, the remaining two pack into the
// variadic slice.
func LeadingFixedThenVariadic() int {
	return sum(threeInts())
}

func oneAfterScale() (int, int) {
	return 10, 7
}

// Exactly one variadic value remains after the fixed parameter.
func SingleVariadicElement() int {
	return sum(oneAfterScale())
}

var sideEffectCalls int

func countedResults() (int, int, int) {
	sideEffectCalls++
	return 1, sideEffectCalls, sideEffectCalls
}

// The inner call must evaluate exactly once.
func InnerEvaluatedOnce() (int, int) {
	sideEffectCalls = 0
	got := sum(countedResults())
	return got, sideEffectCalls
}
`)
}

// A truly empty variadic tail: g returns exactly the fixed parameters, so
// the variadic parameter is the nil slice (rest == nil), never a fresh
// non-nil empty slice.
func TestOracleTupleVariadicEmptyTail(t *testing.T) {
	runOracle(t, `package fixture

func receiveTwo(a int, b int, rest ...int) (int, bool) {
	return len(rest), rest == nil
}

func pair() (int, int) {
	return 1, 2
}

func EmptyTailIsNil() (int, bool) {
	return receiveTwo(pair())
}

func receiveOne(a int, rest ...int) (int, bool) {
	return len(rest), rest == nil
}

func single() int {
	// single is not multi-value; direct call still yields a nil tail.
	return 5
}

func OneFixedNoTail() (int, bool) {
	return receiveOne(single())
}
`)
}

// Pure variadic target: no fixed parameters, every g result packs into
// the slice. Zero, one, and many results.
func TestOracleTupleVariadicPure(t *testing.T) {
	runOracle(t, `package fixture

func collect(rest ...int) (int, bool) {
	return len(rest), rest == nil
}

func two() (int, int) {
	return 1, 2
}

func three() (int, int, int) {
	return 7, 8, 9
}

func PureTwoResults() (int, bool) {
	return collect(two())
}

func PureThreeResults() int {
	n, _ := collect(three())
	return n
}
`)
}

// The variadic element type differs from the source slot type, so each
// remaining result converts exactly like an ordinary argument would.
func TestOracleTupleVariadicConversion(t *testing.T) {
	runOracle(t, `package fixture

func joinStrings(sep string, parts ...string) string {
	out := ""
	for i, s := range parts {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

func threeStrings() (string, string, string) {
	return "-", "a", "b"
}

func StringVariadicForwarding() string {
	return joinStrings(threeStrings())
}
`)
}

// Struct-valued variadic elements copy per slot (a later mutation of one
// packed element never reaches another), and each is a distinct instance.
func TestOracleTupleVariadicStructCopy(t *testing.T) {
	runOracle(t, `package fixture

type pt struct {
	x int
}

func firstX(items ...pt) int {
	if len(items) == 0 {
		return -1
	}
	items[0].x = 999 // mutate the packed copy
	return items[0].x
}

func twoPts() (pt, pt) {
	return pt{x: 1}, pt{x: 2}
}

func base() pt {
	return pt{x: 5}
}

// The source values must not be aliased by the packed slice.
func StructVariadicCopies() (int, int) {
	a := base()
	got := firstX(a, a)
	return got, a.x
}

func PackedTwoStructs() int {
	return firstX(twoPts())
}
`)
}
