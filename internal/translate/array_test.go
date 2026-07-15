// Fixed-array contract tests: differential oracles for value copies,
// in-place whole-value stores, storage-sharing slice views, bounds
// panics, equality, and range snapshots — plus fail-closed diagnostics
// for the array classes still outside the reviewed subset.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleArrayValueSemantics(t *testing.T) {
	runOracle(t, `package fixture

func CopyOnAssign() (int, int) {
	a := [3]int{1, 2, 3}
	b := a
	b[0] = 99
	return a[0], b[0]
}

func mutate(a [3]int) {
	a[0] = -1
}

func CopyOnCall() int {
	a := [3]int{5, 6, 7}
	mutate(a)
	return a[0]
}

func source() [2]int {
	return [2]int{10, 20}
}

func CopyOnReturn() (int, int) {
	a := source()
	b := source()
	a[1] = 42
	return a[1], b[1]
}

func ZeroValue() (int, bool, bool) {
	var nums [3]int
	var flags [2]bool
	var names [2]string
	return nums[2], flags[0], names[1] == ""
}

func LiteralTailZeros() (int, int) {
	a := [4]int{8}
	return a[0], a[3]
}

func Length() int {
	a := [5]int{}
	return len(a) + int(len(a))
}
`)
}

func TestOracleArrayStoresViewsAndAliases(t *testing.T) {
	runOracle(t, `package fixture

func ElementStoreSeenByView() (int, int) {
	a := [3]int{1, 2, 3}
	s := a[:]
	a[1] = 42
	s[2] = 7
	return s[1], a[2]
}

func WholeStoreSeenByView() int {
	a := [3]int{1, 2, 3}
	s := a[:]
	a = [3]int{7, 8, 9}
	return s[0] + s[1] + s[2]
}

func PartialViewBounds() (int, int, int) {
	a := [5]int{1, 2, 3, 4, 5}
	s := a[1:3]
	return len(s), cap(s), s[1]
}

func ViewAppendWithinCapacity() (int, int) {
	a := [4]int{1, 2, 3, 4}
	s := a[0:2]
	s = append(s, 99)
	return a[2], s[2]
}

type pair struct {
	x int
	y int
}

func StructElementsCopyDistinctly() (int, int) {
	a := [2]pair{{x: 1}, {x: 2}}
	b := a
	b[0].x = 50
	return a[0].x, b[0].x
}

func NestedArrayCopy() (int, int) {
	a := [2][2]int{{1, 2}, {3, 4}}
	b := a
	b[1][0] = 30
	return a[1][0], b[1][0]
}

type holder struct {
	vals [3]int
}

func ArrayFieldCopiesWithStruct() (int, int) {
	h := holder{vals: [3]int{1, 2, 3}}
	g := h
	g.vals[0] = 77
	return h.vals[0], g.vals[0]
}

func ArrayFieldWholeStore() int {
	h := holder{}
	h.vals = [3]int{4, 5, 6}
	return h.vals[0] + h.vals[2]
}
`)
}

func TestOracleArrayEqualityRangeAndPanics(t *testing.T) {
	runOracle(t, `package fixture

func EqualAndNot() (bool, bool, bool) {
	a := [3]int{1, 2, 3}
	b := [3]int{1, 2, 3}
	c := [3]int{1, 2, 4}
	return a == b, a == c, b != c
}

func StringArrayEquality() bool {
	a := [2]string{"go", "ts"}
	b := [2]string{"go", "ts"}
	return a == b
}

func RangeSnapshot() int {
	a := [4]int{1, 2, 3, 4}
	total := 0
	for i, v := range a {
		if i == 0 {
			a[3] = 100
		}
		total += v
	}
	return total + a[3]
}

func RangeIndexOnly() int {
	a := [3]int{5, 6, 7}
	total := 0
	for i := range a {
		total += i
	}
	return total
}

func BoundsPanic() int {
	a := [3]int{1, 2, 3}
	i := 3
	return a[i]
}

func StorePanic() int {
	a := [2]int{}
	i := -1
	a[i] = 5
	return a[0]
}

type triple [3]int

func (t triple) sum() int {
	t[0] = 1000
	return t[0] + t[1] + t[2]
}

func NamedArrayValueReceiver() (int, int) {
	v := triple{1, 2, 3}
	got := v.sum()
	return got, v[0]
}
`)
}

func TestArrayFailClosedClasses(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		mention string
	}{
		{
			name: "keyed literal",
			source: `package fixture

func Case() int {
	a := [4]int{2: 9}
	return a[2]
}
`,
			mention: "keyed array literal",
		},
		{
			name: "append of array elements",
			source: `package fixture

func Case() int {
	var s [][2]int
	s = append(s, [2]int{1, 2})
	return len(s)
}
`,
			mention: "append of fixed-array elements",
		},
		{
			name: "copy of array elements",
			source: `package fixture

func Case() int {
	src := [][2]int{{1, 2}}
	dst := make([][2]int, 1)
	return copy(dst, src)
}
`,
			mention: "copy of fixed-array elements",
		},
		{
			name: "equality on struct elements",
			source: `package fixture

type p struct{ x int }

func Case() bool {
	var a, b [2]p
	return a == b
}
`,
			mention: "equality on array of",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": c.source})
			if err == nil {
				t.Fatalf("expected a fail-closed diagnostic mentioning %q", c.mention)
			}
			if !strings.Contains(err.Error(), c.mention) {
				t.Fatalf("expected diagnostic to mention %q, got: %v", c.mention, err)
			}
		})
	}
}
