package calibration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripHeaderRemovesOnlyTheLeadingCommentBlock(t *testing.T) {
	source := "// header line one\n// header line two\n\nfunc x() {\n  // in-body comment stays\n}\n"
	got := string(stripHeader([]byte(source)))
	want := "func x() {\n  // in-body comment stays\n}\n"
	if got != want {
		t.Fatalf("stripHeader = %q; want %q", got, want)
	}
}

func TestCountTokensSkipsCommentsAndCountsStringsOnce(t *testing.T) {
	// return "a b c"; -> return, string, semicolon = 3 tokens; the
	// trailing comment counts zero.
	got := countTokens([]byte("return \"a b c\"; // not tokens\n"))
	if got != 3 {
		t.Fatalf("countTokens = %d; want 3", got)
	}
}

func TestMeasureJoinsAuthoredAndPendingRows(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "F1.ts"), []byte("// header\nab cd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Fixtures: []Fixture{
		{FixtureID: "F1", GoBytes: 6, BaselineArtifactBytes: 30, CandidateVerdict: "ordinary"},
		{FixtureID: "F2", GoBytes: 10, BaselineArtifactBytes: 20, CandidateVerdict: "exception"},
	}}
	report, err := Measure(manifest, dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.FixturesAuthored != 1 || len(report.Summary.FixturesPending) != 1 ||
		report.Summary.FixturesPending[0] != "F2" {
		t.Fatalf("summary = %+v", report.Summary)
	}
	authored := report.Fixtures[0]
	if authored.Status != "authored" || authored.HandPortBytes != 6 || authored.HandPortRatio != 1.0 {
		t.Fatalf("authored row = %+v", authored)
	}
	if report.Fixtures[1].Status != "pending" || report.Fixtures[1].GeneratedRatio != 2.0 {
		t.Fatalf("pending row = %+v", report.Fixtures[1])
	}
	// The single authored ordinary fixture is the median.
	if report.Summary.OrdinaryHandPortMedian != 1.0 {
		t.Fatalf("median = %v", report.Summary.OrdinaryHandPortMedian)
	}
}

func TestMeasureEvenCountMedianAveragesTheMiddlePair(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"G1.ts": "aa\n",    // 3 bytes / 3 -> 1.0
		"G2.ts": "aaaaa\n", // 6 bytes / 3 -> 2.0
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := &Manifest{Fixtures: []Fixture{
		{FixtureID: "G1", GoBytes: 3, CandidateVerdict: "ordinary"},
		{FixtureID: "G2", GoBytes: 3, CandidateVerdict: "ordinary"},
	}}
	report, err := Measure(manifest, dir)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.OrdinaryHandPortMedian != 1.5 {
		t.Fatalf("even-count median = %v; want 1.5", report.Summary.OrdinaryHandPortMedian)
	}
}
