// Multi-result boundary tests from review round five: forwarding,
// returning, and assigning tuple results must apply the same per-slot
// interface boxing (and struct copy) that a single assignment would.
package translate_test

import "testing"

func TestOracleTupleInterfaceForwarding(t *testing.T) {
	runOracle(t, `package fixture

type box struct{ n int }

func (b box) get() int { return b.n }

type getter interface{ get() int }

func pair() (box, int) { return box{n: 7}, 2 }

func take(g getter, n int) int { return g.get() + n }

func ForwardIntoInterfaceCall() int {
	return take(pair())
}

func forwardReturn() (getter, int) {
	return pair()
}

func ReturnForwardBoxes() int {
	g, n := forwardReturn()
	return g.get() + n
}

func AssignForwardBoxes() (int, int) {
	var g getter
	var n int
	g, n = pair()
	return g.get(), n
}

func DeclForwardBoxes() (int, int) {
	g, n := forwardReturn()
	return g.get(), n
}

func triple() (box, box, string) {
	return box{n: 1}, box{n: 2}, "s"
}

func takeThree(a, b getter, s string) int {
	return a.get() + b.get() + len(s)
}

func MultiSlotBoxing() int {
	return takeThree(triple())
}
`)
}

func TestOracleTupleCommaOkBoxing(t *testing.T) {
	runOracle(t, `package fixture

type box struct{ n int }

func (b box) get() int { return b.n }

type getter interface{ get() int }

func CommaOkIntoInterface() (int, bool) {
	m := map[string]box{"a": {n: 5}}
	var g getter
	var ok bool
	g, ok = m["a"]
	return g.get(), ok
}

func CommaOkMissingBoxesZero() (int, bool) {
	m := map[string]box{}
	g, ok := m["absent"]
	var iface getter = g
	return iface.get(), ok
}
`)
}
