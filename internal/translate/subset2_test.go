// Second-subset differential oracles: defer through try/finally, structs
// with pointer receivers, nil semantics, and maps with exact
// nil/zero/comma-ok behavior — each proven byte-identical against Go.
package translate_test

import "testing"

func TestOracleDefer(t *testing.T) {
	runOracle(t, `package fixture

type Trace struct {
	Log int64
}

func mark(trace *Trace, value int64) {
	trace.Log = trace.Log*10 + value
}

func lifoBody(trace *Trace) {
	defer mark(trace, 1)
	defer mark(trace, 2)
	mark(trace, 3)
}

func DeferLIFO() int64 {
	trace := &Trace{Log: 0}
	lifoBody(trace)
	return trace.Log
}

func afterReturn(trace *Trace) int64 {
	defer mark(trace, 9)
	return trace.Log
}

func DeferRunsAfterResultEvaluation() (int64, int64) {
	trace := &Trace{Log: 5}
	returned := afterReturn(trace)
	return returned, trace.Log
}

func captureAtDeferSite(trace *Trace) {
	value := int64(1)
	defer mark(trace, value)
	value = 8
	mark(trace, value)
}

func DeferCapturesArgumentsAtSite() int64 {
	trace := &Trace{Log: 0}
	captureAtDeferSite(trace)
	return trace.Log
}

func (t *Trace) bump() {
	t.Log++
}

func methodDefer(trace *Trace) int64 {
	defer trace.bump()
	return trace.Log * 100
}

func DeferMethodCall() (int64, int64) {
	trace := &Trace{Log: 3}
	scaled := methodDefer(trace)
	return scaled, trace.Log
}
`)
}

func TestOracleStructsAndMethods(t *testing.T) {
	runOracle(t, `package fixture

type Counter struct {
	Total int32
	Label string
	Next  *Counter
}

func (c *Counter) Add(amount int32) int32 {
	c.Total = c.Total + amount
	return c.Total
}

func (c *Counter) Rename(label string) {
	c.Label = label
}

func KeyedAndZeroFields() (int32, string, bool) {
	counter := &Counter{Total: 7}
	return counter.Total, counter.Label, counter.Next == nil
}

func PositionalFields() (int32, string) {
	counter := &Counter{40, "answer", nil}
	counter.Total = counter.Total + 2
	return counter.Total, counter.Label
}

func MethodsMutateReceiver() (int32, int32, string) {
	counter := &Counter{}
	first := counter.Add(5)
	second := counter.Add(6)
	counter.Rename("done")
	return first, second, counter.Label
}

func PointerIdentity() (bool, bool) {
	left := &Counter{Total: 1}
	right := &Counter{Total: 1}
	same := left
	return left == right, left == same
}

func LinkedFields() int32 {
	head := &Counter{Total: 1, Next: &Counter{Total: 2}}
	return head.Total + head.Next.Total
}

func NilDerefPanics() int32 {
	var counter *Counter
	return counter.Total
}

func NilMethodCallPanics() int32 {
	var counter *Counter
	return counter.Add(1)
}
`)
}

func TestOracleMaps(t *testing.T) {
	runOracle(t, `package fixture

func MakeSetGet() (int32, int32) {
	table := make(map[string]int32)
	table["a"] = 41
	table["a"] = table["a"] + 1
	return table["a"], table["missing"]
}

func LiteralAndDelete() (int64, int64, bool) {
	table := map[string]int64{"x": 1, "y": 2, "z": 3}
	before := int64(len(table))
	delete(table, "y")
	delete(table, "absent")
	_, stillThere := table["y"]
	return before, int64(len(table)), stillThere
}

func CommaOkBothBranches() (int32, bool, int32, bool) {
	table := map[int64]int32{10: 100}
	present, okPresent := table[10]
	missing, okMissing := table[11]
	return present, okPresent, missing, okMissing
}

func StoredZeroVersusMissing() (bool, bool) {
	table := make(map[string]int32)
	table["zero"] = 0
	_, hasZero := table["zero"]
	_, hasMissing := table["missing"]
	return hasZero, hasMissing
}

type Node struct {
	ID int32
}

func StoredNilPointerVersusMissing() (bool, bool, bool) {
	table := make(map[string]*Node)
	table["nil"] = nil
	stored, hasStored := table["nil"]
	_, hasMissing := table["missing"]
	return hasStored, stored == nil, hasMissing
}

func NilMapReads() (int32, int64, bool) {
	var table map[string]int32
	value := table["anything"]
	_, ok := table["anything"]
	delete(table, "anything")
	return value, int64(len(table)), ok
}

func NilMapWritePanics() int32 {
	var table map[string]int32
	table["boom"] = 1
	return table["boom"]
}
`)
}

func TestOracleStringLen(t *testing.T) {
	runOracle(t, `package fixture

func ASCIIBytes() int64 {
	return int64(len("hello"))
}

func MultiByteRunes() (int64, int64, int64) {
	twoByte := int64(len("héllo"))
	threeByte := int64(len("中文"))
	fourByte := int64(len("🚀"))
	return twoByte, threeByte, fourByte
}

func EmptyAndConcat() (int64, int64) {
	empty := ""
	combined := "a" + "é"
	return int64(len(empty)), int64(len(combined))
}
`)
}
