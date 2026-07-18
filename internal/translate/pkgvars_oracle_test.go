package translate_test

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/oracle"
)

func oracleRunModule(t *testing.T, packageSources map[string]string) (*oracle.Result, error) {
	t.Helper()
	return oracle.Run(t.TempDir(), packageSources)
}

func assertTranslateFails(t *testing.T, source, code, mention string) {
	t.Helper()
	_, _, err := oracle.Translate(t.TempDir(), map[string]string{"fixture": source})
	if err == nil {
		t.Fatalf("expected fail-closed diagnostic %s", code)
	}
	if !strings.Contains(err.Error(), code) {
		t.Fatalf("expected diagnostic code %s, got: %v", code, err)
	}
	if !strings.Contains(err.Error(), mention) {
		t.Fatalf("expected diagnostic to mention %q, got: %v", mention, err)
	}
}

// helperVarsSource is a co-translated package with package-level state
// read across the unit through live ESM bindings.
const helperVarsSource = `package helper

type Registry struct {
	Count int
	Label string
}

var Shared = &Registry{Count: 0, Label: "shared"}

var Limit = 10

var Names = map[int]string{1: "one", 2: "two"}

var Weights = []int{5, 10, 15}

func Bump(delta int) int {
	Shared.Count = Shared.Count + delta
	return Shared.Count
}

func RaiseLimit(next int) {
	Limit = next
}
`

func TestOraclePackageVariables(t *testing.T) {
	result, err := oracleRunModule(t, map[string]string{
		"fixture": `package fixture

import "oracle.fixture/helper"

var counter int

var table = map[string]int{"a": 1}

var points = []int{1, 2, 3}

var box = helper.Registry{Count: 7, Label: "box"}

func LocalVarState() (int, int) {
	counter = counter + 1
	counter = counter + 2
	table["b"] = counter
	return counter, table["b"]
}

func LocalTables() (int, int) {
	points[0] = points[0] + 10
	return points[0], len(points)
}

func StructVar() (int, string) {
	box.Count = box.Count + 1
	local := box
	local.Count = 99
	return box.Count, box.Label
}

func CrossPackageReads() (int, string, int) {
	return helper.Limit, helper.Names[2], helper.Weights[1]
}

// CrossPackageLiveBindings proves reads observe mutations made by the
// owning package after module initialization.
func CrossPackageLiveBindings() (int, int, int, int) {
	before := helper.Limit
	helper.RaiseLimit(42)
	after := helper.Limit
	first := helper.Bump(3)
	second := helper.Bump(4)
	_ = first
	return before, after, second, helper.Shared.Count
}

// CrossPackageFieldStore mutates a field of a foreign package variable
// (the namespace binding itself is never reassigned).
func CrossPackageFieldStore() string {
	helper.Shared.Label = "renamed"
	return helper.Shared.Label
}

func closureOverVar() func() int {
	return func() int {
		counter = counter + 100
		return counter
	}
}

func ClosureOverPackageVar() (int, int) {
	bump := closureOverVar()
	bump()
	return bump(), counter
}
`,
		"helper": helperVarsSource,
	})
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if !result.Match() {
		t.Fatalf("differential mismatch:\n--- go ---\n%s--- generated ---\n%s", result.GoOutput, result.TSOutput)
	}
}

func TestOraclePointerMapKeys(t *testing.T) {
	runOracle(t, `package fixture

type Node struct {
	ID int
}

func PointerKeys() (int, int, bool, bool, int) {
	a := &Node{ID: 1}
	b := &Node{ID: 1}
	seen := map[*Node]int{}
	seen[a] = 10
	seen[b] = 20
	seen[a] = seen[a] + 1
	_, hasA := seen[a]
	var absent *Node
	_, hasAbsent := seen[absent]
	seen[nil] = 7
	return seen[a], seen[b], hasA, hasAbsent, len(seen)
}
`)
}

func TestOraclePackageVarCallInitializer(t *testing.T) {
	runOracle(t, `package fixture

func produce() int { return 41 }

var v = produce() + 1

func Case() int { return v }
`)
}
