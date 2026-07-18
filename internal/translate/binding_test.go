// Canonical-binding-identity differential proof: a binding that shadows an
// outer one with the same source spelling must keep a distinct identity, so
// a reference to the outer binding (evaluated before the shadow binds) still
// reads the outer value. These fail while variables are keyed by spelling
// (outer i and inner i collapse to one TS identifier) and pass once every
// binding carries a canonical identity with a unique emitted name.
package translate_test

import "testing"

func TestOracleForInitShadowsOuter(t *testing.T) {
	// The initializer's RHS `i` is the OUTER i (=7); the loop then declares a
	// fresh i (=1) and j (=7). Go returns 17.
	runOracle(t, `package fixture

func ForInitShadow() int {
	i := 7
	for i, j := 1, i; i == 1 && j == 7; i++ {
		return 10*i + j
	}
	return -1
}
`)
}

func TestOracleBlockShadowsOuter(t *testing.T) {
	// The nested-block equivalent: `i, j := 1, i` binds j to the outer i (=7)
	// and a fresh i (=1). Go returns 17. Proves the fix is language-wide, not
	// loop-specific.
	runOracle(t, `package fixture

func BlockShadow() int {
	i := 7
	{
		i, j := 1, i
		return 10*i + j
	}
}
`)
}

func TestOracleShadowedParameter(t *testing.T) {
	// An inner block shadows the parameter n; the outer n must survive.
	runOracle(t, `package fixture

func shadowParam(n int) int {
	sum := n
	if n > 0 {
		n := -n
		sum += n
	}
	return sum*10 + n
}

func ShadowedParameter() int { return shadowParam(7) }
`)
}

func TestOracleShadowedNamedResult(t *testing.T) {
	// An inner binding named like the result must NOT write the result cell.
	runOracle(t, `package fixture

func namedResultShadow() (r int) {
	{
		r := 99
		_ = r
	}
	return
}

func ShadowedNamedResult() int { return namedResultShadow() + 1 }
`)
}

func TestOracleShadowedRangeVars(t *testing.T) {
	// Nested range loops reuse the same range-variable spelling i.
	runOracle(t, `package fixture

func ShadowedRangeVars() int {
	xs := []int{1, 2}
	ys := []int{10, 20}
	sum := 0
	for _, i := range xs {
		for _, i := range ys {
			sum += i
		}
		sum += i * 100
	}
	return sum
}
`)
}

func TestOracleAddressTakenShadowCells(t *testing.T) {
	// Two address-taken bindings share the spelling x; their cells must be
	// distinct so *p and *q reach different storage.
	runOracle(t, `package fixture

func AddressTakenShadow() int {
	x := 1
	p := &x
	{
		x := 2
		q := &x
		*p = 10
		*q = 20
		return *p*100 + *q + x
	}
}
`)
}

func TestOracleReservedWordShadow(t *testing.T) {
	// A reserved TS word shadowed: escaping and shadow-disambiguation must
	// compose without conflation.
	runOracle(t, `package fixture

func ReservedWordShadow() int {
	in := 1
	sum := in
	{
		in := 2
		sum += in * 10
	}
	return sum
}
`)
}

func TestOracleTypeSwitchShadow(t *testing.T) {
	// Nested type switches reuse the same binding spelling v.
	runOracle(t, `package fixture

func classify(x any) int {
	switch v := x.(type) {
	case int:
		return v * 2
	case string:
		return len(v)
	default:
		return -1
	}
}

func TypeSwitchShadow() int { return classify(21) + classify("abcd") }
`)
}

func TestOracleNestedShadowSlices(t *testing.T) {
	// An inner slice shadows an outer slice: they are distinct regions with
	// distinct representations, and the outer slice survives the inner one.
	runOracle(t, `package fixture

func NestedShadowSlices() int {
	s := []int{1, 2, 3}
	total := 0
	for _, x := range s {
		s := []int{x * 10}
		s = append(s, x)
		total += s[0] + s[1]
	}
	return total + len(s)
}
`)
}

func TestOracleClosureCaptureShadow(t *testing.T) {
	// A closure captures the outer i; a later inner block re-declares i. The
	// closure must keep reading the captured outer binding.
	runOracle(t, `package fixture

func ClosureCaptureShadow() int {
	i := 3
	read := func() int { return i }
	{
		i := 100
		_ = i
	}
	return read()*10 + i
}
`)
}
