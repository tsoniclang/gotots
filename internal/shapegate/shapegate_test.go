package shapegate

import (
	"os"
	"strings"
	"testing"
)

// MUTATION: the current baseline A1 artifact (capability-vector calls)
// must FAIL the ordinary verdict for the measured reason.
func TestCurrentBaselineA1FailsOrdinary(t *testing.T) {
	body, err := os.ReadFile("../../calibration/testdata/A1-baseline.ts.txt")
	if err != nil {
		t.Skip("baseline artifact fixture not staged")
	}
	eval := EvaluateOrdinary("A1", string(body), "regExpParser", "getSpellingSuggestionForUnicodePropertyValue")
	if eval.Verdict != VerdictFail {
		t.Fatalf("baseline A1 must fail ordinary, got %s", eval.Verdict)
	}
	if eval.HiddenOperationArgs == 0 {
		t.Fatal("baseline A1 must show hidden operation arguments")
	}
}

// The A1 hand port passes the same checks.
func TestHandPortA1PassesOrdinary(t *testing.T) {
	body, err := os.ReadFile("../../calibration/handports/A1.ts")
	if err != nil {
		t.Skip("hand port not staged")
	}
	eval := EvaluateOrdinary("A1", string(body), "regExpParser", "getSpellingSuggestionForUnicodePropertyValue")
	if eval.Verdict != VerdictPass {
		t.Fatalf("hand port A1 must pass, got %s: %v", eval.Verdict, eval.Failures)
	}
}

// MUTATION: a free-function method facade is detected.
func TestFacadeDetected(t *testing.T) {
	body := "export function Counter$Add(counter: Counter, amount: bigint): bigint { return 0n; }"
	eval := EvaluateOrdinary("X", body, "Counter", "Add")
	if !eval.FreeFunctionFacade || eval.Verdict != VerdictFail {
		t.Fatalf("facade must fail: %+v", eval)
	}
}

// MUTATION: duplicate alias definitions are counted; references are not.
func TestDuplicateDefinitionsCounted(t *testing.T) {
	modules := map[string]string{
		"a.ts": "type Iface$X = A | B;\nlet v: Iface$X;",
		"b.ts": "type Iface$X = A | B;",
	}
	if d := CountDuplicateDefinitions(modules, "type Iface$X ="); d != 1 {
		t.Fatalf("expected 1 duplicate definition, got %d", d)
	}
}

// MUTATION: a stale fixture hash fails closed.
func TestStaleFixtureHashFails(t *testing.T) {
	if err := VerifyFixtureHash([]byte("body"), strings.Repeat("0", 64)); err == nil {
		t.Fatal("stale hash must fail")
	}
}
