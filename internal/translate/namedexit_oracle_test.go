package translate_test

import "testing"

func TestOracleDeferNamedResults(t *testing.T) {
	// The osvfs.ReadFile shape: defer in a function with named results.
	// Deferred mutations of the named locals must reach the returned
	// values — including the defer-registered-conditionally and
	// multi-result forwarding forms.
	runOracle(t, `package fixture

func release(counter *int) func() {
	*counter++
	return func() {
		*counter++
	}
}

func guarded(counter *int) (contents string, ok bool) {
	defer release(counter)()
	return "data", true
}

func mutated() (n int) {
	defer func() {
		n *= 10
	}()
	n = 4
	return n + 1
}

func bareReturn() (s string) {
	defer func() {
		s += "!"
	}()
	s = "hi"
	return
}

func earlyNoDefer(skip bool) (n int) {
	if skip {
		return 7
	}
	defer func() {
		n += 100
	}()
	return 1
}

func DeferNamedResults() int {
	total := 0
	counter := 0
	contents, ok := guarded(&counter)
	if contents == "data" && ok && counter == 2 {
		total += 1
	}
	if mutated() == 50 {
		total += 10
	}
	if bareReturn() == "hi!" {
		total += 100
	}
	if earlyNoDefer(true) == 7 {
		total += 1000
	}
	if earlyNoDefer(false) == 101 {
		total += 10000
	}
	return total
}
`)
}
