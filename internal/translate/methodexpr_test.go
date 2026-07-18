// Method-expression contract tests from review round five: (*T).M for a
// value-receiver method M takes *T, dereferences (nil panics), and
// copies the value on entry.
package translate_test

import "testing"

func TestOracleMethodExpressionAdapters(t *testing.T) {
	runOracle(t, `package fixture

type counter struct{ n int }

func (c counter) value() int  { return c.n }
func (c *counter) bump()       { c.n++ }

func ValueMethodExpression() int {
	f := counter.value
	c := counter{n: 5}
	return f(c)
}

func PointerMethodExpressionOfValueMethod() int {
	f := (*counter).value
	c := &counter{n: 9}
	return f(c)
}

func PointerMethodExpression() int {
	f := (*counter).bump
	c := &counter{n: 1}
	f(c)
	f(c)
	return c.n
}

func ValueMethodExprCopiesReceiver() int {
	f := (*counter).value
	c := &counter{n: 3}
	got := f(c)
	c.n = 100
	return got
}
`)
}
