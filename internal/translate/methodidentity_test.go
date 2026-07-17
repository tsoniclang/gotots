// Method-slot identity through interface dispatch. Dispatch selects a
// method by its name within a union member narrowed by literal
// discriminant; this is exact because Go permits exactly one method per
// name per type and types.Implements enforces signature identity and
// unexported-method package identity for union membership, and each member
// carries its own vtable object. These cases pin that identity across the
// hardest collisions: unexported methods, a spelling that collides with a
// JavaScript Object.prototype member, and the same unexported spelling in
// two different packages.
package translate_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

// Unexported interface methods and a method whose spelling collides with a
// JS Object.prototype member both dispatch by exact identity.
func TestOracleMethodIdentityDispatch(t *testing.T) {
	runOracle(t, `package fixture

type reader interface{ read() int }
type stringer interface{ toString() string }

type a struct{ n int }

func (x a) read() int        { return x.n }
func (x a) toString() string { return "a" }

type b struct{ n int }

func (x b) read() int        { return x.n * 10 }
func (x b) toString() string { return "b" }

func Dispatch() (int, int, string, string) {
	var r1 reader = a{n: 3}
	var r2 reader = b{n: 3}
	var s1 stringer = a{n: 0}
	var s2 stringer = b{n: 0}
	return r1.read(), r2.read(), s1.toString(), s2.toString()
}
`)
}

// Two packages each define an interface with an unexported method of the
// SAME spelling `tag`. Go's method-set rules make the two identities
// distinct (package-qualified); dispatch in the importer routes each to
// its own member's slot.
func TestOracleCrossPackageUnexportedMethodIdentity(t *testing.T) {
	result, err := oracle.Run(t.TempDir(), map[string]string{
		"fixture": `package fixture

import (
	"oracle.fixture/left"
	"oracle.fixture/right"
)

func Case() (int, int) {
	return left.Tagged(), right.Tagged()
}
`,
		"left": `package left

type tagger interface{ tag() int }
type impl struct{}

func (impl) tag() int { return 1 }

func Tagged() int { var t tagger = impl{}; return t.tag() }
`,
		"right": `package right

type tagger interface{ tag() int }
type impl struct{}

func (impl) tag() int { return 2 }

func Tagged() int { var t tagger = impl{}; return t.tag() }
`,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("mismatch:\n--- go ---\n%s--- gen ---\n%s", result.GoOutput, result.TSOutput)
	}
}
