// Multi-result-into-variadic contract tests: Go's f(g()) where f ends in
// a variadic parameter binds g's leading results to the regular
// parameters and packs the rest into the final slice, evaluating g once.
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

func exactlyOne() (int, int) {
	return 10, 7
}

// Exactly one variadic value remains after the fixed parameter.
func SingleVariadicElement() int {
	return sum(exactlyOne())
}

func onlyFixed() (int, int) {
	// The variadic gets no values: the slice is empty.
	return 5, 100
}

func firstOnly(scale int, rest ...int) int {
	return scale + len(rest)
}

func EmptyVariadicTail() int {
	return firstOnly(onlyFixed())
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

// The variadic element type differs from the source slot type, so each
// remaining result converts (widening int32 results into an int64
// variadic) exactly like an ordinary argument would.
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
