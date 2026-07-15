// Breadth contract tests for the smaller reviewed classes: min/max,
// new(T), string-family conversions, range over maps, compound
// assignment through pure field chains, and := reusing existing names.
package translate_test

import "testing"

func TestOracleMinMaxAndNew(t *testing.T) {
	runOracle(t, `package fixture

func MinMaxInts() (int, int, int32) {
	a, b, c := 3, -7, 3
	return min(a, b, c), max(a, b, c), min(int32(9), int32(4))
}

func MinMaxStringsAndBytes() (string, string, bool) {
	lo := min("pear", "apple", "\xff")
	hi := max("pear", "apple", "\xff")
	return lo, hi, hi == "\xff"
}

func MinMaxFloatZeros() (bool, bool) {
	negZero := math_copysign()
	lo := min(0.0, negZero)
	hi := max(negZero, 0.0)
	return isNegZero(lo), isNegZero(hi)
}

func math_copysign() float64 {
	z := 0.0
	return -z
}

func isNegZero(f float64) bool {
	return f == 0 && 1/f < 0
}

func MinMaxNaN() (bool, bool) {
	nan := notANumber()
	return min(1.0, nan) != min(1.0, nan), max(nan, 2.0) != max(nan, 2.0)
}

func notANumber() float64 {
	z := 0.0
	return z / z
}

type box struct {
	n int
}

func NewStruct() (int, int) {
	p := new(box)
	q := new(box)
	p.n = 5
	return p.n, q.n
}
`)
}

func TestOracleStringConversions(t *testing.T) {
	runOracle(t, `package fixture

func RuneToString() (string, string, string, string) {
	return string(rune(65)), string(rune(0xE9)), string(rune(0x1F600)), string(rune(0xD800))
}

func ByteToString() (string, int) {
	b := byte(0xFF)
	s := string(rune(b))
	return s, len(s)
}

func BytesRoundTrip() (string, int, bool) {
	src := "hé\xff"
	bytes := []byte(src)
	bytes[0] = 'H'
	back := string(bytes)
	return back, len(bytes), src == "hé\xff"
}

func RunesRoundTrip() (string, int, int) {
	src := "a😀\xff"
	runes := []rune(src)
	back := string(runes)
	return back, len(runes), len(back)
}

func NilSlicesConvert() (string, string) {
	var bytes []byte
	var runes []rune
	return string(bytes), string(runes)
}
`)
}

func TestOracleRangeMapAndReuse(t *testing.T) {
	runOracle(t, `package fixture

func RangeMapSum() (int, int) {
	m := map[string]int{"a": 1, "b": 2, "c": 4}
	total, count := 0, 0
	for _, v := range m {
		total += v
		count++
	}
	return total, count
}

func RangeMapKeysSorted() string {
	m := map[string]int{"z": 1, "a": 2, "m": 3}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys[0] + keys[1] + keys[2]
}

func RangeNilMap() int {
	var m map[string]int
	count := 0
	for range m {
		count++
	}
	return count
}

func lookup(m map[string]int, k string) (int, bool) {
	v, ok := m[k]
	return v, ok
}

func ShortDeclReuse() (int, int, bool) {
	m := map[string]int{"x": 10}
	v, ok := lookup(m, "x")
	v2, ok := lookup(m, "missing")
	return v, v2, ok
}
`)
}

func TestOracleCompoundFieldChains(t *testing.T) {
	runOracle(t, `package fixture

type inner struct {
	n int
}

type outer struct {
	in inner
}

type holder struct {
	out outer
}

func CompoundThroughChain() int {
	h := holder{}
	h.out.in.n += 5
	h.out.in.n *= 3
	h.out.in.n++
	return h.out.in.n
}

func CompoundThroughPointerChain() int {
	h := &holder{}
	h.out.in.n += 7
	return h.out.in.n
}

func CompoundNilChainPanics() int {
	var h *holder
	h.out.in.n += 1
	return 0
}
`)
}

func TestOracleDeferredClosuresAndLocalConsts(t *testing.T) {
	runOracle(t, `package fixture

var trace string

func deferredClosure() {
	x := 1
	defer func() {
		trace += "deferred"
	}()
	x = 2
	_ = x
	trace += "body:"
}

func DeferredClosureRuns() string {
	trace = ""
	deferredClosure()
	return trace
}

func LocalConsts() (int, string) {
	const width = 640
	const label = "px"
	const derived = width / 2
	return derived, label
}

func BlankClosureParams() int {
	f := func(_ int, y int, _ string) int {
		return y * 2
	}
	return f(9, 21, "ignored")
}
`)
}
