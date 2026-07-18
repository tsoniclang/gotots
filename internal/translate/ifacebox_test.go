// Composite and external interface-value contract tests: slices, maps,
// funcs, arrays, and external named values boxed into interfaces, with
// interned rtti identity for assertions, type switches, and
// contract-routed method dispatch.
package translate_test

import "testing"

func TestOracleCompositeInterfaceValues(t *testing.T) {
	runOracle(t, `package fixture

func classify(v any) string {
	switch v.(type) {
	case []string:
		return "strings"
	case []int:
		return "ints"
	case map[string]int:
		return "map"
	case [2]int:
		return "array"
	case func() int:
		return "func"
	case nil:
		return "nil"
	}
	return "other"
}

func TypeSwitchOverComposites() (string, string, string, string, string, string) {
	f := func() int { return 1 }
	return classify([]string{"a"}),
		classify([]int{1}),
		classify(map[string]int{}),
		classify([2]int{1, 2}),
		classify(f),
		classify(nil)
}

func AssertBack() (int, bool, bool) {
	var v any = []int{5, 6, 7}
	got := v.([]int)
	_, isStrings := v.([]string)
	_, isInts := v.([]int)
	return len(got), isStrings, isInts
}

func AssertPanicMessage() int {
	var v any = []string{"x"}
	return len(v.([]int))
}

func ArrayValueBoxCopies() (int, int) {
	arr := [2]int{1, 2}
	var v any = arr
	arr[0] = 99
	back := v.([2]int)
	return back[0], arr[0]
}

func SliceIdentityThroughInterface() int {
	s := []int{1, 2, 3}
	var v any = s
	back := v.([]int)
	return int(back[0]) + len(back)
}
`)
}
