// Unnamed / blank receiver and parameter contract tests: a receiver or
// parameter with no name (or the blank identifier) is never referenced
// in the body, so it carries a generated placeholder name and the
// function behaves exactly as Go's.
package translate_test

import "testing"

func TestOracleBlankParameters(t *testing.T) {
	runOracle(t, `package fixture

// Unnamed parameter.
func constFromUnnamed(int) int {
	return 42
}

// Blank parameter between named ones: the middle argument is evaluated
// and discarded, exactly as Go.
func middleBlank(a int, _ string, b int) int {
	return a + b
}

// Two blank parameters need distinct placeholder slots.
func twoBlanks(_ int, _ int) int {
	return 7
}

var evaluated int

func sideEffect() string {
	evaluated++
	return "x"
}

func BlankArgumentStillEvaluated() (int, int) {
	evaluated = 0
	got := middleBlank(1, sideEffect(), 2)
	return got, evaluated
}

func UnnamedAndTwoBlanks() (int, int) {
	return constFromUnnamed(99), twoBlanks(1, 2)
}
`)
}

func TestOracleBlankReceivers(t *testing.T) {
	runOracle(t, `package fixture

type widget struct {
	id int
}

// Unnamed pointer receiver.
func (*widget) kind() int {
	return 1
}

// Blank value receiver.
func (widget) tag() string {
	return "w"
}

func ReceiversWork() (int, string) {
	w := &widget{id: 5}
	return w.kind(), w.tag()
}

type calc struct {
	base int
}

// Named and blank receivers on the same type coexist.
func (c calc) withName() int {
	return c.base
}

func (calc) withoutName() int {
	return 100
}

func MixedReceivers() (int, int) {
	c := calc{base: 9}
	return c.withName(), c.withoutName()
}
`)
}
