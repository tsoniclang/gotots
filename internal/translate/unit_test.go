// Unit-type contract tests: the anonymous empty struct struct{} carries
// as the literal 0 — set-valued maps, unit literals, equality, and
// zero values are all exact.
package translate_test

import "testing"

func TestOracleUnitType(t *testing.T) {
	runOracle(t, `package fixture

func SetMembership() (bool, bool, int) {
	seen := map[string]struct{}{}
	seen["a"] = struct{}{}
	seen["b"] = struct{}{}
	_, hasA := seen["a"]
	_, hasC := seen["c"]
	return hasA, hasC, len(seen)
}

func SetLiteral() (bool, int) {
	seen := map[int]struct{}{1: {}, 2: {}}
	_, has := seen[2]
	delete(seen, 1)
	return has, len(seen)
}

func UnitEquality() (bool, bool) {
	a := struct{}{}
	b := struct{}{}
	return a == b, a != b
}

func unitArg(v struct{}) int {
	_ = v
	return 7
}

func UnitParamAndZero() int {
	var z struct{}
	return unitArg(z)
}
`)
}
