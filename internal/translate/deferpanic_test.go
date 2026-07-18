// Defer, panic, and payload contract tests from review round five:
// deferred panics continue draining older defers, deferred value
// receivers copy at the defer statement, and panic payloads format
// lazily (after defers run).
package translate_test

import "testing"

func TestOracleDeferredPanicDrain(t *testing.T) {
	runOracle(t, `package fixture

var trace string

func drain() {
	defer func() { trace += "A;" }()
	defer func() { trace += "B;"; panic("replacement") }()
	trace += "body;"
}

func ADrainPanics() int {
	trace = ""
	drain()
	return 0
}

func ZReadTrace() string {
	return trace
}

var order string

func layered() {
	defer func() { order += "outer;" }()
	defer func() { order += "middle;"; panic("second") }()
	defer func() { order += "inner;" }()
	panic("first")
}

func BLayeredPanics() int {
	order = ""
	layered()
	return 0
}

func YReadOrder() string {
	return order
}
`)
}

func TestOracleDeferredReceiverCopy(t *testing.T) {
	runOracle(t, `package fixture

type counter struct{ n int }

var saved int

func (c counter) record() { saved = c.n }

func run() {
	p := &counter{n: 1}
	defer p.record()
	p.n = 2
}

func ValueReceiverCopiesAtDefer() int {
	saved = 0
	run()
	return saved
}

func runValue() {
	c := counter{n: 5}
	defer c.record()
	c.n = 6
}

func ValueReceiverLocalCopiesAtDefer() int {
	saved = 0
	runValue()
	return saved
}

func runNil() {
	var p *counter
	defer func() { recover_placeholder() }()
	defer p.record()
	_ = p
}

func recover_placeholder() {}

func NilPointerValueReceiverPanicsAtDefer() int {
	saved = 0
	runNil()
	return saved
}
`)
}

func TestOracleLazyPanicPayload(t *testing.T) {
	runOracle(t, `package fixture

type stamp struct{ msg string }

func (s *stamp) Error() string { return s.msg }

func run() {
	err := &stamp{msg: "before"}
	defer func() { err.msg = "after" }()
	panic(err)
}

func LazyPanicMessage() int {
	run()
	return 0
}
`)
}
