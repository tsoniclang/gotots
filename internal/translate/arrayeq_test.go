// Recursive interface equality: comparable arrays boxed in any (including
// nested arrays and arrays of comparable structs) compare element-wise
// through one recursive typed equality plan, never a flat mode that
// falsely reports the dynamic type uncomparable.
package translate_test

import "testing"

type eqPt struct {
	x int
	y int
}

func TestOracleArrayInterfaceEquality(t *testing.T) {
	runOracle(t, `package fixture

type pt struct {
	x int
	y int
}

func FlatArray() (bool, bool) {
	var a any = [2]int{1, 2}
	var b any = [2]int{1, 2}
	var c any = [2]int{1, 3}
	return a == b, a == c
}

func NestedArray() (bool, bool) {
	var a any = [2][2]int{{1, 2}, {3, 4}}
	var b any = [2][2]int{{1, 2}, {3, 4}}
	var c any = [2][2]int{{1, 2}, {9, 9}}
	return a == b, a == c
}

func ArrayOfStruct() (bool, bool) {
	var a any = [2]pt{{1, 2}, {3, 4}}
	var b any = [2]pt{{1, 2}, {3, 4}}
	var c any = [2]pt{{1, 2}, {9, 9}}
	return a == b, a == c
}

func ArrayOfPointer() bool {
	x, y := 1, 1
	var a any = [1]*int{&x}
	var b any = [1]*int{&y}
	return a == b
}
`)
}
