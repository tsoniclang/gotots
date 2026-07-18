// Range-over-func contract tests: iterator sequences with break,
// continue, early return (single and multi-result), nested loops and
// switches inside the body, nested sequences, and the exact panic when
// a sequence keeps yielding after the loop exits.
package translate_test

import "testing"

func TestOracleRangeFunc(t *testing.T) {
	runOracle(t, `package fixture

func upTo(n int) func(func(int) bool) {
	return func(yield func(int) bool) {
		for i := 0; i < n; i++ {
			if !yield(i) {
				return
			}
		}
	}
}

func pairs(yield func(string, int) bool) {
	if !yield("a", 1) {
		return
	}
	if !yield("b", 2) {
		return
	}
	yield("c", 3)
}

func SumAll() int {
	total := 0
	for v := range upTo(5) {
		total += v
	}
	return total
}

func BreakStops() int {
	total := 0
	for v := range upTo(100) {
		if v == 3 {
			break
		}
		total += v
	}
	return total
}

func ContinueSkips() int {
	total := 0
	for v := range upTo(6) {
		if v%2 == 0 {
			continue
		}
		total += v
	}
	return total
}

func EarlyReturn() int {
	for v := range upTo(10) {
		if v == 4 {
			return v * 100
		}
	}
	return -1
}

func EarlyReturnMulti() (int, string) {
	for k, v := range pairs {
		if v == 2 {
			return v * 10, k
		}
	}
	return 0, "none"
}

func TwoVarIteration() (string, int) {
	keys := ""
	total := 0
	for k, v := range pairs {
		keys += k
		total += v
	}
	return keys, total
}

func NestedLoopBreakInside() int {
	total := 0
	for v := range upTo(3) {
		for i := 0; i < 10; i++ {
			if i == 2 {
				break
			}
			total += v + i
		}
	}
	return total
}

func SwitchInsideBody() int {
	total := 0
	for v := range upTo(5) {
		switch v {
		case 1:
			total += 10
		case 3:
			continue
		default:
			total++
		}
	}
	return total
}

func NestedSequencesEarlyReturn() int {
	for a := range upTo(3) {
		for b := range upTo(3) {
			if a+b == 3 {
				return a*10 + b
			}
		}
	}
	return -1
}

func misbehaving(yield func(int) bool) {
	yield(1)
	yield(2)
	yield(3)
}

func YieldAfterBreakPanics() int {
	total := 0
	for v := range misbehaving {
		total += v
		break
	}
	return total
}
`)
}
