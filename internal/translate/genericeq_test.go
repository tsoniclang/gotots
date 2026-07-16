// Generic type-parameter operation tests from review round five: == over
// a type parameter uses the instantiation's exact equality — direct ===
// for comparable scalars, interface equality (with the uncomparable
// panic) for interface instantiations.
package translate_test

import "testing"

func TestOracleGenericEquality(t *testing.T) {
	runOracle(t, `package fixture

func eq[T comparable](a, b T) bool { return a == b }

func neq[T comparable](a, b T) bool { return a != b }

func IntEquality() (bool, bool) {
	return eq(3, 3), eq(3, 4)
}

func StringEquality() (bool, bool) {
	return eq("go", "go"), neq("go", "ts")
}

func InterfaceEqualitySameType() (bool, bool) {
	var a any = 5
	var b any = 5
	var c any = 6
	return eq(a, b), eq(a, c)
}

func InterfaceEqualityDifferentDynamicType() bool {
	var a any = 5
	var b any = "5"
	return eq(a, b)
}

func InterfaceEqualityPointers() (bool, bool) {
	x, y := 1, 1
	var a any = &x
	var b any = &x
	var c any = &y
	return eq(a, b), eq(a, c)
}

func UncomparableInterfacePanics() bool {
	m := map[string]int{"k": 1}
	var a any = m
	var b any = m
	return eq(a, b)
}
`)
}
