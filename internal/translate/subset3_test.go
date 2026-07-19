package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleSwitch(t *testing.T) {
	runOracle(t, `package fixture

func classify(x int) string {
	switch x {
	case 1, 2:
		return "small"
	case 3:
		return "three"
	default:
		return "other"
	}
}

func Multi() (string, string, string, string) {
	return classify(1), classify(2), classify(3), classify(99)
}

func Fallthrough() int {
	total := 0
	switch 2 {
	case 1:
		total += 1
		fallthrough
	case 2:
		total += 10
		fallthrough
	case 3:
		total += 100
	case 4:
		total += 1000
	}
	return total
}

func FallthroughIntoDefault() int {
	total := 0
	switch 1 {
	case 1:
		total += 1
		fallthrough
	default:
		total += 10
	}
	return total
}

func DefaultInMiddle() int {
	switch 42 {
	case 1:
		return 1
	default:
		return -1
	case 42:
		return 2
	}
}

func tagless(x int) int {
	switch {
	case x < 0:
		return -1
	case x == 0:
		return 0
	default:
		return 1
	}
}

func TaglessAll() (int, int, int) {
	return tagless(-5), tagless(0), tagless(7)
}

func WithInit() int {
	switch x := 6 * 7; x {
	case 42:
		return x
	}
	return 0
}

func CaseScopes() (int, int) {
	a, b := 0, 0
	switch 1 {
	case 1:
		x := 10
		a = x
	}
	switch 2 {
	case 2:
		x := 20
		b = x
	}
	return a, b
}

func BreakInSwitchInLoop() (int, int) {
	iterations, hits := 0, 0
	for i := 0; i < 5; i++ {
		iterations++
		switch i {
		case 2:
			hits++
			break
		case 3:
			continue
		}
		hits += 100
	}
	return iterations, hits
}

func stringSwitch(s string) int {
	switch s {
	case "alpha":
		return 1
	case "beta", "gamma":
		return 2
	}
	return 3
}

func Strings() (int, int, int, int) {
	return stringSwitch("alpha"), stringSwitch("beta"), stringSwitch("gamma"), stringSwitch("delta")
}

func Int64Switch() string {
	var x int64 = 1 << 40
	switch x {
	case 1 << 40:
		return "big"
	}
	return "no"
}

type tracker struct {
	Count int
}

func bump(t *tracker, value int) int {
	t.Count++
	return value
}

// SideEffectOrder proves case expressions evaluate in source order and
// stop at the first match on both sides.
func SideEffectOrder() (int, int) {
	t := &tracker{Count: 0}
	order := 0
	switch 2 {
	case bump(t, 1):
		order = 1
	case bump(t, 2):
		order = 2
	case bump(t, 3):
		order = 3
	}
	return order, t.Count
}
`)
}

func TestOracleCompoundTargets(t *testing.T) {
	runOracle(t, `package fixture

type Counter struct {
	Total int
	Bits  uint32
	Label string
}

func FieldCompound() (int, uint32, string) {
	c := &Counter{Total: 40, Bits: 1, Label: "x"}
	c.Total += 2
	c.Total *= 3
	c.Bits <<= 4
	c.Bits |= 3
	c.Label += "yz"
	c.Total++
	c.Total--
	return c.Total, c.Bits, c.Label
}

func NilFieldCompound() int {
	var c *Counter
	c.Total += 1
	return c.Total
}

func MapCompound() (int, int, int) {
	m := map[string]int{"a": 10}
	m["a"] += 5
	m["missing"] += 7
	m["a"]++
	m["b"]--
	return m["a"], m["missing"], m["b"]
}

func NilMapCompound() int {
	var m map[string]int
	m["k"] += 1
	return m["k"]
}

func SliceCompound() (int, int) {
	s := []int{1, 2, 3}
	s[0] += 10
	s[1] *= 5
	s[2]++
	return s[0] + s[1] + s[2], len(s)
}

func SliceCompoundOutOfRange() int {
	s := []int{1}
	i := 3
	s[i] += 1
	return s[i]
}
`)
}

func TestOracleNamedResults(t *testing.T) {
	runOracle(t, `package fixture

func split(total int) (low int, high int) {
	low = total % 10
	high = total / 10
	return
}

func Bare() (int, int) {
	return split(47)
}

func zeroDefault() (count int, label string) {
	if false {
		count = 99
	}
	return
}

func Zeros() (int, string) {
	return zeroDefault()
}

func mixedReturn(flag bool) (value int) {
	if flag {
		return 7
	}
	value = 3
	return
}

func Mixed() (int, int) {
	return mixedReturn(true), mixedReturn(false)
}

func named64() (total int64) {
	for i := int64(0); i < 5; i++ {
		total += i
	}
	return
}

func Accumulate() int64 {
	return named64()
}
`)
}

func TestOracleVariadic(t *testing.T) {
	runOracle(t, `package fixture

func sum(values ...int) (int, int) {
	total := 0
	for _, v := range values {
		total += v
	}
	return total, len(values)
}

func Pack() (int, int) {
	return sum(1, 2, 3)
}

func Empty() (int, int) {
	return sum()
}

func Spread() (int, int) {
	s := []int{10, 20, 30}
	total, count := sum(s...)
	return total, count
}

func prefixed(scale int, values ...int) int {
	total := 0
	for _, v := range values {
		total += v * scale
	}
	return total
}

func Fixed() (int, int) {
	return prefixed(10, 1, 2), prefixed(3)
}

func mutate(values ...int) {
	if len(values) > 0 {
		values[0] = 999
	}
}

func SpreadAliases() int {
	s := []int{1, 2}
	mutate(s...)
	return s[0]
}

func PackCopies() int {
	x := 1
	mutate(x, 2)
	return x
}

type Acc struct {
	Total int
}

func (a *Acc) Add(values ...int) int {
	for _, v := range values {
		a.Total += v
	}
	return a.Total
}

func VariadicMethod() (int, int) {
	a := &Acc{Total: 0}
	a.Add(1, 2, 3)
	s := []int{10, 20}
	return a.Add(s...), a.Total
}

func DeferredVariadic() int {
	a := &Acc{Total: 0}
	x := 5
	defer a.Add(x, 10)
	x = 99
	a.Add(1)
	return a.Total
}
`)
}

func TestNamedResultsWithDeferTranslate(t *testing.T) {
	// The named-exit lowering admits defers in top-level named-results
	// functions (deferred mutations reach the returned values — the
	// differential oracle covers the semantics).
	source := `package fixture

func Case() (value int) {
	defer helper()
	value = 1
	return
}

func helper() {}
`
	if _, err := oracle.Run(t.TempDir(), map[string]string{"fixture": source}); err != nil {
		t.Fatalf("named-results defer should translate: %v", err)
	}
}
