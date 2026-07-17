// Return-boundary behavior for named results. A bare `return` yields the
// named result variables (they ARE the result storage; the caller copies).
// An explicit `return expr` copies value structs into the result. The one
// case where these could diverge — a deferred function observing and
// mutating the named result after an explicit return assigns it — is
// fail-closed, so no return path miscompiles.
package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// A value-struct named result yields a distinct instance per call; mutating
// one caller's copy never reaches another's, on both the bare-return and
// explicit-return paths.
func TestOracleNamedResultStructCopy(t *testing.T) {
	runOracle(t, `package fixture

type P struct{ x int }

func makeP(v int) (p P) {
	p.x = v
	return
}

func explicitP(v int) P {
	q := P{x: v}
	return q
}

func Case() (int, int, int, int) {
	a := makeP(1)
	b := makeP(2)
	a.x = 99
	c := explicitP(3)
	d := c
	c.x = 77
	return a.x, b.x, c.x, d.x
}
`)
}

// Deferred mutation of a named result would make an explicit `return expr`
// observably return the defer-updated value; that interaction is
// fail-closed rather than silently returning the pre-defer value.
func TestNamedResultDeferMutationWithheld(t *testing.T) {
	_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": `package fixture

func doubled() (r int) {
	defer func() { r *= 2 }()
	return 5
}
`})
	if err == nil {
		t.Fatal("expected deferred named-result mutation to be withheld")
	}
	if !strings.Contains(err.Error(), "defer in a function with named results") {
		t.Fatalf("expected the deferred-result-mutation block, got: %v", err)
	}
}
