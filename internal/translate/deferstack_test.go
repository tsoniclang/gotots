// Defer-stack contract tests: defers below the top level run at
// function exit in LIFO order interleaved with top-level defers,
// capture their arguments at the defer site, run per loop iteration,
// and fire on panic unwinds.
package translate_test

import "testing"

func TestOracleDeferStack(t *testing.T) {
	runOracle(t, `package fixture

var log string

func record(tag string) {
	log += tag
}

func conditionalDefer(enable bool) {
	log += "["
	if enable {
		defer record("c")
	}
	log += "-"
}

func ConditionalDefers() (string, string) {
	log = ""
	conditionalDefer(true)
	first := log
	log = ""
	conditionalDefer(false)
	return first, log
}

func loopDefers() {
	for i := 0; i < 3; i++ {
		defer record(string(rune('a' + i)))
	}
	log += "|"
}

func LoopDefersLIFO() string {
	log = ""
	loopDefers()
	return log
}

func mixedDefers() {
	defer record("T")
	if true {
		defer record("n")
	}
	log += "b"
}

func MixedTopAndNestedLIFO() string {
	log = ""
	mixedDefers()
	return log
}

func capturesAtSite() {
	x := 1
	if true {
		defer record(string(rune('0' + x)))
	}
	x = 7
	_ = x
}

func CapturesArgsAtDeferSite() string {
	log = ""
	capturesAtSite()
	return log
}

func panicking() {
	for i := 0; i < 2; i++ {
		defer record("p")
	}
	panic("boom")
}

func DefersRunOnPanic() (string, string) {
	log = ""
	defer record("outer")
	panicking()
	return "unreached", ""
}

func closureDefers() string {
	log = ""
	f := func() {
		if true {
			defer record("in")
		}
		log += "x"
	}
	f()
	log += "y"
	return log
}

func ClosureOwnsItsDefers() string {
	return closureDefers()
}
`)
}
