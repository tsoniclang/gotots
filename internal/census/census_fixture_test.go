// Census contract fixtures: semantic identity, unique declaration IDs,
// tracked-universe classification, external attribution, contradiction
// recording, and clean-run determinism.
package census

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCensusFixture(t *testing.T) {
	dir, revision := writeFixtureRepo(t, basicFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	// Ambient toolchain/workspace/module state must not influence the run.
	t.Setenv("GOWORK", filepath.Join(dir, "no-such-workspace"))
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOTOOLCHAIN", "go1.1.1")

	result, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	report := result.Report

	// Black-box test files keep their semantic package identity while being
	// owned by the package under test.
	var blackBox *FileRecord
	for i := range report.Files {
		if report.Files[i].Path == "a/a_bb_test.go" {
			blackBox = &report.Files[i]
		}
	}
	if blackBox == nil {
		t.Fatal("black-box test file missing from report")
	}
	if blackBox.Package != "example.com/fix/a_test" || blackBox.Owner != "example.com/fix/a" || blackBox.Scope != "test" {
		t.Errorf("black-box identity wrong: %+v", blackBox)
	}

	// Legal duplicate declarations receive position-qualified unique IDs.
	blanks, inits := 0, 0
	for _, d := range report.Declarations {
		if d.File == "a/a.go" && d.Name == "_" {
			blanks++
			if !strings.Contains(d.ID, "::var::_@") {
				t.Errorf("blank var ID not position-qualified: %s", d.ID)
			}
		}
		if d.File == "a/a.go" && d.Name == "init" {
			inits++
			if !strings.Contains(d.ID, "::func::init@") {
				t.Errorf("init ID not position-qualified: %s", d.ID)
			}
		}
	}
	if blanks != 2 || inits != 2 {
		t.Errorf("expected 2 blank vars and 2 inits, got %d and %d", blanks, inits)
	}

	// A legitimate package path ending in .test is ordinary owned source.
	foundBTest := false
	for _, f := range report.Files {
		if f.Package == "example.com/fix/b.test" && f.Scope == "production" {
			foundBTest = true
		}
	}
	if !foundBTest {
		t.Error("package with path ending in .test was not analyzed as production source")
	}

	// Universe: tool-module and testdata files are classified, none lost,
	// module metadata enumerated, and the total reconciles.
	universe := result.Inventory.Universe
	classes := map[string]string{}
	for _, f := range universe.GoOutsidePackages {
		classes[f.Path] = f.Class
	}
	if classes["_tools/gen/main.go"] != "tooling" {
		t.Errorf("nested tool module file not classified tooling: %v", classes)
	}
	if classes["a/testdata/fixture.go"] != "testdata" {
		t.Errorf("testdata file not classified: %v", classes)
	}
	metadata := strings.Join(universe.ModuleMetadata, ",")
	if !strings.Contains(metadata, "go.mod") || !strings.Contains(metadata, "_tools/gen/go.mod") {
		t.Errorf("module metadata incomplete: %v", universe.ModuleMetadata)
	}
	accounted := universe.InPackages + len(universe.GoOutsidePackages) +
		len(universe.ModuleMetadata) + universe.NonBuildFiles
	if universe.TrackedFiles != accounted {
		t.Errorf("universe does not reconcile: tracked=%d accounted=%d (%+v)", universe.TrackedFiles, accounted, universe)
	}

	// External attribution: os is a test-only dependency (via support);
	// net/http is reachable only through the hard-excluded package. The
	// -deps contract must supply transitive test-dependency evidence (io is
	// in os's closure).
	externals := map[string]*ExternalUseEvidence{}
	for i := range result.Inventory.External {
		e := &result.Inventory.External[i]
		externals[e.ImportPath] = &ExternalUseEvidence{
			Production: e.ReachableFromProduction, Test: e.ReachableFromTest, ExcludedOnly: e.ExcludedOrUnselectedOnly,
		}
	}
	if e := externals["os"]; e == nil || e.Production || !e.Test {
		t.Errorf("os should be test-only reachable: %+v", e)
	}
	if e := externals["io"]; e == nil || !e.Test {
		t.Errorf("io (transitive test dependency) missing or misattributed: %+v", e)
	}
	if e := externals["net/http"]; e == nil || !e.ExcludedOnly {
		t.Errorf("net/http should be excluded-only: %+v", e)
	}

	// The production import of a hard-excluded package is a recorded
	// contradiction, not a silent reclassification.
	foundEdge := false
	for _, edge := range report.Contradictions {
		if edge.From == "example.com/fix/c" && edge.To == "example.com/fix/ex" &&
			edge.Class == "hard-excluded" && edge.Scope == "production" && edge.File == "c/c.go" {
			foundEdge = true
		}
	}
	if !foundEdge {
		t.Errorf("expected contradiction edge for c -> ex, got %+v", report.Contradictions)
	}

	// Publication is transactional and deterministic.
	out1 := filepath.Join(t.TempDir(), "reports")
	if err := WriteReports(result, out1); err != nil {
		t.Fatalf("write reports: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out1, "manifest.json")); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	second, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("second census: %v", err)
	}
	out2 := filepath.Join(t.TempDir(), "reports")
	if err := WriteReports(second, out2); err != nil {
		t.Fatalf("write second reports: %v", err)
	}
	for _, name := range []string{"inventory.json", "census.json"} {
		first, err := os.ReadFile(filepath.Join(out1, name))
		if err != nil {
			t.Fatal(err)
		}
		again, err := os.ReadFile(filepath.Join(out2, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(first) != string(again) {
			t.Errorf("%s is not deterministic across clean runs", name)
		}
	}
}

// ExternalUseEvidence is a test-local view of external attribution.
type ExternalUseEvidence struct {
	Production   bool
	Test         bool
	ExcludedOnly bool
}
