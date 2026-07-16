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

func TestOracleVariadicFunctionValues(t *testing.T) {
	runOracle(t, `package fixture

func sum(base int, values ...int) int {
	total := base
	for _, v := range values {
		total += v
	}
	return total
}

func VariadicFuncValue() (int, int, int) {
	f := sum
	direct := f(10, 1, 2, 3)
	empty := f(5)
	spread := f(0, []int{4, 5}...)
	return direct, empty, spread
}

func VariadicClosure() int {
	join := func(sep string, parts ...string) string {
		out := ""
		for i, part := range parts {
			if i > 0 {
				out += sep
			}
			out += part
		}
		return out
	}
	g := join
	return len(g("-", "a", "b", "c"))
}
`)
}

func TestOracleTupleForwarding(t *testing.T) {
	runOracle(t, `package fixture

func pair() (int, string) {
	return 7, "seven"
}

func consume(n int, s string) string {
	out := s
	for i := 0; i < n; i++ {
		out += "!"
	}
	return out
}

func ForwardTuple() string {
	return consume(pair())
}
`)
}

func TestOracleFloatToIntConversions(t *testing.T) {
	runOracle(t, `package fixture

func values() []float64 {
	inf := 1.0
	zero := 0.0
	return []float64{2.9, -2.9, 0.0, 1e30, -1e30, inf / zero, -inf / zero, zero / zero}
}

func FloatToInt64() string {
	out := ""
	for _, f := range values() {
		out += " "
		v := int64(f)
		if v == -9223372036854775808 {
			out += "MIN"
		} else {
			out += string(rune('0' + v%10))
		}
	}
	return out
}

func FloatToInt() (int, int) {
	a := 7.99
	b := -7.99
	return int(a), int(b)
}

func FloatToInt32() (int32, int32, int32, int32) {
	vs := values()
	return int32(vs[0]), int32(vs[1]), int32(vs[3]), int32(vs[7])
}
`)
}

func TestOracleStagedCompoundTargets(t *testing.T) {
	runOracle(t, `package fixture

type node struct {
	weight int
	next   *node
}

func chain() *node {
	return &node{weight: 1, next: &node{weight: 10}}
}

var calls int

func pick(n *node) *node {
	calls++
	return n.next
}

func CompoundThroughCallBase() (int, int) {
	calls = 0
	n := chain()
	pick(n).weight += 5
	return n.next.weight, calls
}

func CompoundOnMapMissing() (int, int) {
	m := map[string]int{}
	m["k"] += 3
	m["k"] *= 4
	return m["k"], len(m)
}

func indexOnce() func() int {
	i := 0
	return func() int {
		i++
		return i - 1
	}
}

func CompoundSliceImpureIndex() (int, int, int) {
	next := indexOnce()
	s := []int{10, 20}
	s[next()] += 7
	s[next()] += 9
	return s[0], s[1], 2
}

func CompoundIncDecPointee() int {
	x := 5
	p := &x
	*p += 2
	*p++
	return x
}
`)
}

func TestOracleElidedPointerLiterals(t *testing.T) {
	runOracle(t, `package fixture

type entry struct {
	code int
	name string
}

var registry = map[int]*entry{
	1: {code: 1, name: "one"},
	2: {code: 2, name: "two"},
}

var flat = []*entry{
	{code: 10, name: "ten"},
	{code: 20, name: "twenty"},
}

func ElidedAllocations() (string, int, string) {
	registry[1].name = "uno"
	return registry[1].name, flat[1].code, flat[0].name
}
`)
}

func TestOraclePlannedNativeSlices(t *testing.T) {
	runOracle(t, `package fixture

func OwnerOnlySlice() (int, int, int) {
	values := []int{1, 2}
	values = append(values, 3)
	values = append(values, 4, 5)
	values[0] = 10
	total := 0
	for _, v := range values {
		total += v
	}
	return total, len(values), values[4]
}

func OwnerOnlyMake() (int, int) {
	buf := make([]int, 3)
	buf[1] = 7
	buf = append(buf, 9)
	return len(buf), buf[1] + buf[3]
}

func OwnerOnlyBounds() int {
	values := []int{1}
	i := 5
	return values[i]
}

func CarrierWhenNilObserved() bool {
	var s []int
	s = append(s, 1)
	return s == nil
}

func CarrierWhenShared() (int, int) {
	base := []int{1, 2, 3, 4}
	view := base[1:3]
	view[0] = 99
	return base[1], len(view)
}
`)
}

func TestOracleCompoundEvaluationOrder(t *testing.T) {
	runOracle(t, `package fixture

var mapVar map[string]int
var sliceVar []int
var ptrVar *int
var fieldPtr *holder2
var callCount int

type holder2 struct{ v int }

func rebindMapRHS() int {
	fresh := map[string]int{"k": 100}
	mapVar = fresh
	return 2
}

func MapContainerReadAfterRHS() int {
	mapVar = map[string]int{"k": 1}
	mapVar["k"] += rebindMapRHS()
	return mapVar["k"]
}

func rebindSliceRHS() int {
	sliceVar = []int{0, 100}
	return 2
}

func SliceHeaderReadAfterRHS() int {
	sliceVar = []int{0, 1}
	sliceVar[1] += rebindSliceRHS()
	return sliceVar[1]
}

func rebindPtrRHS() int {
	fresh := 100
	ptrVar = &fresh
	return 7
}

func PointerReadAfterRHS() int {
	base := 1
	ptrVar = &base
	*ptrVar += rebindPtrRHS()
	return *ptrVar
}

func rebindFieldPtr() int {
	fresh := holder2{v: 700}
	fieldPtr = &fresh
	return 7
}

func FieldBaseReadAfterRHS() int {
	base := holder2{v: 1}
	fieldPtr = &base
	fieldPtr.v += rebindFieldPtr()
	return fieldPtr.v
}

func lexicalKey() string {
	callCount = 1
	return "a"
}

func lexicalRHS() int {
	callCount = callCount*10 + 2
	return 5
}

func KeyThenRHSLexicalOrder() (int, int) {
	m := map[string]int{"a": 20}
	callCount = 0
	m[lexicalKey()] += lexicalRHS()
	return m["a"], callCount
}
`)
}

func TestOracleRangeFuncInvalidation(t *testing.T) {
	runOracle(t, `package fixture

func seq(yield func(int) bool) {
	yield(1)
	yield(2)
	yield(3)
}

func NormalIteration() int {
	total := 0
	for v := range seq {
		total += v
	}
	return total
}

func BreakStopsIteration() int {
	total := 0
	for v := range seq {
		total += v
		if v == 2 {
			break
		}
	}
	return total
}

func NilSequencePanics() int {
	var s func(func(int) bool)
	count := 0
	for range s {
		count++
	}
	return count
}

func ReturnFromRange() int {
	for v := range seq {
		if v == 2 {
			return v * 10
		}
	}
	return -1
}
`)
}
