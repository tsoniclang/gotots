// Interface-to-interface assertion narrows the SOURCE union: only types
// implementing both source and target are candidates, so the literal
// discriminant checks stay within the source union's members (never
// comparing the source's k against a target-only implementer's literal).
package translate_test

import "testing"

func TestOracleAssertionSourceTargetIntersection(t *testing.T) {
	runOracle(t, `package fixture

type Reader interface{ Read() int }
type Writer interface{ Write() int }

type onlyR struct{}
func (onlyR) Read() int { return 1 }

type onlyW struct{}
func (onlyW) Write() int { return 2 }

type rw struct{}
func (rw) Read() int  { return 3 }
func (rw) Write() int { return 4 }

func AssertMiss() (int, bool) {
	var r Reader = onlyR{}
	w, ok := r.(Writer)
	if ok {
		return w.Write(), true
	}
	var keep Writer = onlyW{}
	_ = keep
	return 0, false
}

func AssertHit() (int, bool) {
	var r Reader = rw{}
	w, ok := r.(Writer)
	if ok {
		return w.Write(), true
	}
	return -1, false
}
`)
}
