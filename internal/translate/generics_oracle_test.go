package translate_test

import "testing"

func TestOracleGenerics(t *testing.T) {
	runOracle(t, `package fixture

func mapSlice[T any, U any](values []T, f func(T) U) []U {
	var out []U
	for _, v := range values {
		out = append(out, f(v))
	}
	return out
}

func filter[T any](values []T, keep func(T) bool) []T {
	var out []T
	for _, v := range values {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func contains[T comparable](values []T, target T) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func firstOr[T any](values []T, fallback T) T {
	if len(values) > 0 {
		return values[0]
	}
	return fallback
}

func MapAndFilter() (int, int, string) {
	numbers := []int{1, 2, 3, 4}
	doubled := mapSlice(numbers, func(x int) int { return x * 2 })
	big := filter(doubled, func(x int) bool { return x > 4 })
	words := mapSlice(numbers, func(x int) string {
		if x%2 == 0 {
			return "even"
		}
		return "odd"
	})
	return len(big), big[0], words[1]
}

func Contains() (bool, bool, bool) {
	return contains([]int{1, 2, 3}, 2),
		contains([]string{"a", "b"}, "z"),
		contains([]int64{1 << 40}, 1<<40)
}

type Node struct {
	ID int
}

func ContainsPointers() (bool, bool) {
	a := &Node{ID: 1}
	b := &Node{ID: 1}
	nodes := []*Node{a}
	return contains(nodes, a), contains(nodes, b)
}

func FirstOr() (int, string) {
	return firstOr([]int{7, 8}, -1), firstOr(nil, "empty")
}

func ExplicitInstantiation() int {
	return firstOr[int](nil, 42)
}

func pair[A any, B any](a A, b B) (A, B) {
	return a, b
}

func MultiParamGeneric() (string, int) {
	s, n := pair("x", 9)
	return s, n
}

func nested[T any](values []T) int {
	return len(filter(values, func(v T) bool { return true }))
}

func NestedGenericCalls() int {
	return nested([]string{"a", "b", "c"})
}
`)
}

func TestOracleGenericsCrossPackage(t *testing.T) {
	result, err := oracleRunModule(t, map[string]string{
		"helper": `package helper

func Reverse[T any](values []T) []T {
	var out []T
	for i := len(values) - 1; i >= 0; i-- {
		out = append(out, values[i])
	}
	return out
}

func IndexOf[T comparable](values []T, target T) int {
	for i, v := range values {
		if v == target {
			return i
		}
	}
	return -1
}
`,
		"fixture": `package fixture

import "oracle.fixture/helper"

func CrossPackageGeneric() (string, int, int) {
	words := []string{"a", "b", "c"}
	reversed := helper.Reverse(words)
	return reversed[0], helper.IndexOf(words, "c"), helper.IndexOf(words, "zz")
}
`,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestGenericStructInstantiationFailsClosed(t *testing.T) {
	assertTranslateFails(t, `package fixture

type Point struct {
	X int
}

func identity[T any](v T) T {
	return v
}

func Case() int {
	p := identity(Point{X: 1})
	return p.X
}
`, "GOTOTS_UNSUPPORTED_DECLARATION", "instantiated with a struct value")
}
