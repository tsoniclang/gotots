package translate_test

import "testing"

// These fixtures are the adversarial scenarios from external review round
// four: every one was an accepted-but-wrong translation. Each now either
// matches Go byte-for-byte or fails closed at translation.

func TestReviewRegressionEvaluationOrder(t *testing.T) {
	runOracle(t, `package fixture

type counter struct {
	N int
}

func (c *counter) next() int {
	c.N = c.N + 1
	return c.N
}

func MakeLengthEvaluatesOnce() (int, int, int) {
	c := &counter{}
	s := make([]int, c.next())
	return c.N, len(s), cap(s)
}

func resliceSource(c *counter) []int {
	c.next()
	return []int{10, 20, 30}
}

func ResliceOperandEvaluatesOnce() (int, int, int) {
	c := &counter{}
	a := resliceSource(c)[1:]
	b := resliceSource(c)[:]
	return c.N, len(a), len(b)
}

type pair struct {
	A int
	B int
}

func KeyedLiteralSourceOrder() (int, int) {
	c := &counter{}
	p := &pair{B: c.next(), A: c.next()}
	return p.A, p.B
}

func KeyedLiteralValueForm() (int, int) {
	c := &counter{}
	p := pair{B: c.next(), A: c.next()}
	return p.A, p.B
}

type trace struct {
	Log string
}

func (t *trace) key() string {
	t.Log = t.Log + "K"
	return "a"
}

func (t *trace) idx() int {
	t.Log = t.Log + "I"
	return 0
}

func (t *trace) pair() (int, bool) {
	t.Log = t.Log + "R"
	return 7, true
}

func TupleAssignTargetsFirst() (string, int, bool) {
	tr := &trace{}
	m := map[string]int{"a": 1}
	var ok bool
	m[tr.key()], ok = tr.pair()
	return tr.Log, m["a"], ok
}

func TupleAssignSliceTarget() (string, int) {
	tr := &trace{}
	s := []int{0}
	var x bool
	s[tr.idx()], x = tr.pair()
	_ = x
	return tr.Log, s[0]
}

func panickyArg() int {
	panic("arg evaluated first")
}

func NilFuncCallArgsBeforePanic() int {
	var f func(int) int
	return f(panickyArg())
}
`)
}

func TestReviewRegressionSliceIdentity(t *testing.T) {
	runOracle(t, `package fixture

func AppendNothingKeepsNil() (bool, bool) {
	var s []int
	s = append(s)
	var t []int
	var empty []int
	t = append(t, empty...)
	return s == nil, t == nil
}

func AppendNothingKeepsValue() (bool, int) {
	s := []int{1}
	s2 := append(s)
	return s2 == nil, len(s2)
}

type Point struct {
	X int
	Y int
}

func StructAppendGrowthCopies() (int, int) {
	s := make([]Point, 1, 1)
	s[0].X = 1
	grown := append(s, Point{X: 2, Y: 2})
	s[0].X = 99
	return grown[0].X, s[0].X
}

func GrownCapacityReadsZeros() int {
	s := make([]int, 1, 1)
	grown := append(s, 5)
	extended := grown[:cap(grown)]
	total := 0
	for _, v := range extended {
		total = total + v
	}
	return total
}
`)
}

func TestReviewRegressionRangeVariables(t *testing.T) {
	runOracle(t, `package fixture

func RangeIntBodyAssignment() (int, int) {
	count := 0
	last := 0
	for i := range 5 {
		if i == 1 {
			i = 99
		}
		last = i
		count++
	}
	return count, last
}

func RangeSliceBodyAssignment() (int, int) {
	values := []int{10, 20, 30}
	count := 0
	total := 0
	for i, v := range values {
		if i == 0 {
			i = 50
		}
		total = total + v + i
		count++
	}
	return count, total
}
`)
}

func TestReviewRegressionDeferredReceiverCopy(t *testing.T) {
	runOracle(t, `package fixture

type holder struct {
	N int
}

type rec struct {
	X    int
	sink *holder
}

func (r rec) record() {
	r.sink.N = r.X
}

func DeferValueReceiverCopiesAtDefer() int {
	h := &holder{}
	run := func() {
		v := rec{X: 1, sink: h}
		defer v.record()
		v.X = 99
	}
	run()
	return h.N
}
`)
}

func TestReviewRegressionNameHygiene(t *testing.T) {
	runOracle(t, `package fixture

func ShadowAbiAliases() int32 {
	goabi := int32(3)
	gort := int32(4)
	gosl := int32(5)
	return goabi * gort * gosl
}

func ShadowTemporaries() (int, int) {
	_t0 := 1
	_d0_a0 := 2
	a, b := _d0_a0, _t0
	a, b = b, a
	return a, b
}

func ShadowReservedWords() (int, bool) {
	eval := 6
	static := 7
	var undefined *holder2
	let := eval * static
	return let, undefined == nil
}

type holder2 struct {
	N int
}
`)
}

func TestReviewRegressionFailClosed(t *testing.T) {
	assertTranslateFails(t, `package fixture

var flag int

func init() {
	flag = 41
}

func Case() int { return flag }
`, "GOTOTS_UNSUPPORTED_DECLARATION", "package init function")

	// Non-UTF-8 string constants are exact under the byte-string carrier;
	// their differential coverage lives in the string oracle fixtures.
}
