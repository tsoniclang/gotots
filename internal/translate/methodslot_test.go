// Method-slot identity end to end. A concrete type may promote two DISTINCT
// package-private methods that share a bare name from different packages;
// Go admits both in its method set (each satisfies its own package's
// interface) even though the selector expression is ambiguous. The
// generated vtable gives them DISTINCT slots and each interface dispatches
// to exactly its own method — no bare-name collision, no fail-closed
// rejection.
package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func TestOracleTwoPackagePrivateMethodSlots(t *testing.T) {
	result, err := oracle.Run(t.TempDir(), map[string]string{
		"fixture": `package fixture

import (
	"oracle.fixture/left"
	"oracle.fixture/right"
)

type Both struct {
	left.LeftT
	right.RightT
}

func Case() (int, int) {
	var b Both
	return left.Call(b), right.Call(b)
}
`,
		"left": `package left

type LeftT struct{}

func (LeftT) tag() int { return 111 }

type Tagger interface{ tag() int }

func Call(t Tagger) int { return t.tag() }
`,
		"right": `package right

type RightT struct{}

func (RightT) tag() int { return 222 }

type Tagger interface{ tag() int }

func Call(t Tagger) int { return t.tag() }
`,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}
