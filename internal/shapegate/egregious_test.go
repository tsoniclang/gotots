package shapegate

import "testing"

// Every measured baseline egregious case MUST fail unconditionally.
func TestMeasuredBaselineCasesFail(t *testing.T) {
	fixtures := []FixtureMeasurement{
		// A1: ordinary 286 B -> 1,006 B (3.52x).
		{FixtureID: "A1", Verdict: "ordinary", GoBytes: 286, GeneratedBytes: 1006},
		// D3: 424 B -> 59,612 B (140x) — exception verdict, still fails.
		{FixtureID: "D3", Verdict: "exception", GoBytes: 424, GeneratedBytes: 59612},
	}
	copies := map[string]int{
		// The 661 KB empty-interface union alias in 101 modules.
		"iface::empty-union": 101,
		// One inline vtable identity materialized 3,897 times.
		"vtable::goIfaceBox-inline": 3897,
	}
	failures := EgregiousReport(fixtures, copies)
	want := map[string]bool{
		"egregious-ordinary-expansion/A1":                            false,
		"egregious-expansion/D3":                                     false,
		"egregious-definition-duplication/iface::empty-union":        false,
		"egregious-definition-duplication/vtable::goIfaceBox-inline": false,
	}
	for _, failure := range failures {
		key := failure.Class + "/" + failure.ID
		if _, tracked := want[key]; tracked {
			want[key] = true
		}
	}
	for key, hit := range want {
		if !hit {
			t.Fatalf("measured baseline case did not fail: %s (failures: %v)", key, failures)
		}
	}
}

// Near-1x ordinary output and single definitions pass — the bounds are
// egregious-only, never a calibrated budget.
func TestSourceShapedOutputPasses(t *testing.T) {
	fixtures := []FixtureMeasurement{
		{FixtureID: "A1", Verdict: "ordinary", GoBytes: 286, GeneratedBytes: 293},       // 1.02x
		{FixtureID: "D1", Verdict: "ordinary", GoBytes: 318107, GeneratedBytes: 335582}, // 1.05x
		{FixtureID: "B8", Verdict: "exception", GoBytes: 100, GeneratedBytes: 1020},     // 10.2x exception: below the any-bound
	}
	copies := map[string]int{"iface::union-x": 1}
	if failures := EgregiousReport(fixtures, copies); len(failures) != 0 {
		t.Fatalf("source-shaped output must pass the egregious bounds: %v", failures)
	}
}

// The ordinary bound is stricter than the any-fixture bound.
func TestOrdinaryBoundIsStricter(t *testing.T) {
	if failure := EgregiousExpansion("X", "ordinary", 100, 300); failure == nil {
		t.Fatal("3.0x ordinary must fail the 2.5x unconditional bound")
	}
	if failure := EgregiousExpansion("X", "exception", 100, 300); failure != nil {
		t.Fatalf("3.0x exception is below the any-bound and must pass: %v", failure)
	}
	if failure := EgregiousExpansion("X", "exception", 100, 2100); failure == nil {
		t.Fatal("21x exception must fail the any-bound")
	}
}

// Zero-byte sources never divide.
func TestZeroGoBytesIsNotEvaluated(t *testing.T) {
	if failure := EgregiousExpansion("X", "ordinary", 0, 1000); failure != nil {
		t.Fatalf("zero-byte source must not evaluate: %v", failure)
	}
}
