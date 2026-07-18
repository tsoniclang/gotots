// Method-reference contract tests: method values capture (and copy)
// their receiver at evaluation; method expressions are the bare
// receiver-first function; interface method values dispatch dynamically
// and nil-panic at evaluation.
package translate_test

import "testing"

func TestOracleMethodValues(t *testing.T) {
	runOracle(t, `package fixture

type counter struct {
	n int
}

func (c counter) get() int {
	return c.n
}

func (c *counter) bump() {
	c.n = c.n + 1
}

func ValueReceiverCapturesCopy() (int, int) {
	c := counter{n: 1}
	f := c.get
	c.n = 99
	return f(), c.get()
}

func PointerReceiverSeesMutation() (int, int) {
	c := counter{n: 1}
	bump := (&c).bump
	get := c.get
	bump()
	bump()
	return c.n, get()
}

func PointerCallerValueReceiverCopiesAtBind() int {
	c := &counter{n: 5}
	f := c.get
	c.n = 50
	return f()
}

func NilPointerValueReceiverPanicsAtBind() int {
	var c *counter
	f := c.get
	return f()
}

type triple [3]int

func (t triple) sum() int {
	return t[0] + t[1] + t[2]
}

func ArrayReceiverCapturesCopy() (int, int) {
	v := triple{1, 2, 3}
	f := v.sum
	v[0] = 100
	return f(), v.sum()
}

func MethodValueEvaluatesReceiverOnce() int {
	calls := 0
	make := func() counter {
		calls++
		return counter{n: calls * 10}
	}
	f := make().get
	_ = f()
	_ = f()
	return f() + calls
}
`)
}

func TestOracleMethodExpressions(t *testing.T) {
	runOracle(t, `package fixture

type counter struct {
	n int
}

func (c counter) get() int {
	return c.n
}

func (c *counter) add(delta int) {
	c.n = c.n + delta
}

func ValueMethodExpression() (int, int) {
	f := counter.get
	c := counter{n: 7}
	got := f(c)
	c.n = 70
	return got, f(c)
}

func PointerMethodExpression() int {
	f := (*counter).add
	c := counter{n: 1}
	f(&c, 41)
	return c.n
}

func MethodExpressionCopiesValueReceiver() int {
	f := counter.get
	c := counter{n: 3}
	before := f(c)
	c.n = 4
	return before + f(c)
}
`)
}

func TestOracleInterfaceMethodValues(t *testing.T) {
	runOracle(t, `package fixture

type speaker interface {
	speak() string
}

type dog struct {
	name string
}

func (d *dog) speak() string {
	return "woof:" + d.name
}

func InterfaceMethodValue() (string, string) {
	var s speaker = &dog{name: "rex"}
	f := s.speak
	first := f()
	return first, f()
}

func NilInterfaceMethodValuePanicsAtEvaluation() string {
	var s speaker
	f := s.speak
	return f()
}
`)
}
