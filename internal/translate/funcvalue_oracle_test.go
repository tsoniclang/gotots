package translate_test

import "testing"

func TestOracleFuncValues(t *testing.T) {
	runOracle(t, `package fixture

func double(x int) int {
	return x * 2
}

func apply(f func(int) int, x int) int {
	return f(x)
}

func FuncRefAsValue() int {
	return apply(double, 21)
}

func ClosureLiteral() int {
	add := func(a int, b int) int {
		return a + b
	}
	return add(40, 2)
}

func ClosureCapturesByReference() (int, int) {
	total := 0
	bump := func(delta int) {
		total = total + delta
	}
	bump(5)
	bump(7)
	seen := total
	total = 100
	return seen, total
}

func LoopVariableCapture() (int, int, int) {
	var fns []func() int
	for i := 0; i < 3; i++ {
		fns = append(fns, func() int { return i })
	}
	return fns[0](), fns[1](), fns[2]()
}

func NilFuncComparison() (bool, bool) {
	var f func(int) int
	g := double
	return f == nil, g != nil
}

func NilFuncCallPanics() int {
	var f func(int) int
	return f(1)
}

func FuncReturningFunc() int {
	scale := func(factor int) func(int) int {
		return func(x int) int { return x * factor }
	}
	return scale(3)(14)
}

func ImmediateInvocation() int {
	return func() int { return 7 }()
}

func MultiResultClosure() (int, string) {
	pair := func() (int, string) {
		return 4, "four"
	}
	return pair()
}

func closureWithNamedResult() func() (total int) {
	return func() (total int) {
		total = 9
		return
	}
}

func NamedResultInClosure() int {
	return closureWithNamedResult()()
}

// DeferredFuncValue proves the deferred function value and its
// arguments are captured at the defer site: later reassignments of
// either never reach the deferred call.
func DeferredFuncValue() (int, int) {
	c := &Holder{N: 0}
	run := func() int {
		f := func(delta int) { c.N = c.N + delta }
		x := 5
		defer f(x)
		x = 90
		f = func(delta int) { c.N = c.N + 1000*delta }
		return x
	}
	seen := run()
	return c.N, seen
}

type Holder struct {
	N int
}

func FuncField() int {
	h := &Handler{Fn: double, Name: "d"}
	viaField := h.Fn(10)
	h.Fn = func(x int) int { return x + 1 }
	return viaField + h.Fn(10)
}

type Handler struct {
	Fn   func(int) int
	Name string
}

func NilFuncFieldZero() bool {
	h := &Handler{Name: "empty"}
	return h.Fn == nil
}

func VisitorPattern() int {
	values := []int{1, 2, 3, 4}
	total := 0
	visit := func(v int) bool {
		total = total + v
		return v < 3
	}
	for _, v := range values {
		if !visit(v) {
			break
		}
	}
	return total
}

func ClosureMutatesCapturedStruct() (int, int) {
	v := Holder{N: 1}
	set := func(n int) {
		v.N = n
	}
	set(42)
	w := v
	set(7)
	return v.N, w.N
}
`)
}
