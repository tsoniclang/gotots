// Executable proof for multi-result for-loop clause lowering: every case
// is strict-typechecked with the pinned tsc (catching a mutable-tuple
// annotation over a readonly tuple source, TS4104) AND differentially
// executed Go-vs-generated-TS (catching a lost value copy, a double
// right-hand evaluation, or a post clause that does not run on continue).
// A regression in readonly typing, copying, one-shot evaluation, or
// post-on-continue makes one of these cases fail — they are the mutation
// tests for the loop-lowering contract.
package translate_test

import "testing"

func TestOracleForLoopMultiResultInitAndPost(t *testing.T) {
	// The reported scanner shape: multi-result init AND multi-result post.
	// step returns a readonly tuple in the generated TS, so binding it in a
	// mutable-tuple annotation would fail strict tsc (TS4104).
	runOracle(t, `package fixture

func step(i int) (int, int) { return i + 1, i * i }

func ScannerShape() int {
	sum := 0
	for v, sq := step(0); v <= 5; v, sq = step(v) {
		sum += sq
	}
	return sum
}
`)
}

func TestOracleForLoopOneEvaluation(t *testing.T) {
	// The post's right side must evaluate exactly once per iteration; a
	// double evaluation advances the call counter differently in TS than
	// Go and the differential diverges.
	runOracle(t, `package fixture

func OneEvaluation() int {
	calls := 0
	adv := func(i int) (int, int) { calls++; return i + 1, i }
	last := 0
	for v, w := adv(0); v <= 4; v, w = adv(v) {
		last = w
	}
	return calls*1000 + last
}
`)
}

func TestOracleForLoopContinueRunsPost(t *testing.T) {
	// continue must still run the post exactly once, so v keeps advancing
	// and the loop terminates with the Go-observed sum. If the post were
	// skipped on continue the TS loop would diverge (or not terminate).
	runOracle(t, `package fixture

func ContinueRunsPost() int {
	sum := 0
	posts := 0
	adv := func(i int) (int, int) { posts++; return i + 1, i }
	for v, w := adv(-1); v <= 6; v, w = adv(v) {
		if w%2 == 0 {
			continue
		}
		sum += w
	}
	return posts*1000 + sum
}
`)
}

func TestOracleForLoopParallelSwap(t *testing.T) {
	// A parallel post assignment swaps two locals: every right side must
	// evaluate before any store (Go's two-phase rule).
	runOracle(t, `package fixture

func ParallelSwap() int {
	a, b := 1, 2
	acc := 0
	for i := 0; i < 5; i, a, b = i+1, b, a {
		acc += a*10 + b
	}
	return acc*100 + a*10 + b
}
`)
}

func TestOracleForLoopCommaOkStructInit(t *testing.T) {
	// A comma-ok map read in the loop init binds a struct value: the value
	// carrier must be cloned each entry, not aliased to the map's storage.
	runOracle(t, `package fixture

type point struct{ X, Y int }

func CommaOkStructInit() int {
	m := map[int]point{7: {3, 4}}
	total := 0
	for p, ok := m[7]; ok; ok = false {
		p.X += 100
		total += p.X*10 + p.Y
	}
	// The map entry must be unchanged (p was a copy).
	got := m[7]
	return total*1000 + got.X*10 + got.Y
}
`)
}

func TestOracleForLoopBlankSlotInit(t *testing.T) {
	// A blank result slot in the loop init and post evaluates its source
	// but binds nothing; the surviving variable still advances.
	runOracle(t, `package fixture

func step2(i int) (int, int) { return i + 1, i }

func BlankSlot() int {
	sum := 0
	for v, _ := step2(0); v <= 4; v, _ = step2(v) {
		sum += v
	}
	return sum
}
`)
}
