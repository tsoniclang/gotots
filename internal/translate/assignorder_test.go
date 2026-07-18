// Assignment evaluation order: Go's two-phase rule evaluates the
// right-hand side before carrying out the store, so a nil pointer
// dereference in the target panics AFTER a side-effecting right-hand side
// has run. (*p).field = f() must match p.field = f().
package translate_test

import "testing"

func TestOracleAssignmentStoreOrder(t *testing.T) {
	runOracle(t, `package fixture

type box struct {
	n int
}

var effect int

func sideEffect() int {
	effect++
	return 7
}

// AA-prefixed cases run first (sorted), panic through the nil store, then
// the ZZ reader observes whether the right-hand side ran.
func AAImplicitDeref() int {
	var p *box
	p.n = sideEffect()
	return 0
}

func AAExplicitDeref() int {
	var p *box
	(*p).n = sideEffect()
	return 0
}

func ZZEffectCount() int { return effect }
`)
}
