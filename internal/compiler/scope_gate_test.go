package compiler

import (
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// TestRetainedScopeExactJoin is the scope/cost gate's deterministic half:
// the retained inventory exact-joins the derived full-semantic evidence set
// (both one-sided checks), no declaration-contract file contributes a row,
// and the retained-occurrence count sits inside the reviewed budget frozen
// from three isolated clean runs (webshop 469 retained at calibration;
// budget leaves headroom for fixture growth, never for a scope-class
// regression: the pre-partition value was 598,836).
func TestRetainedScopeExactJoin(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "testdata", "projects", "webshop")
	inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	expected := map[string]bool{}
	for _, pkg := range inspection.Workspace().Packages() {
		for _, file := range pkg.Files() {
			switch evidence := file.Evidence().(type) {
			case source.FullSyntax:
				expected[file.ID().String()] = true
			case source.MixedUnits:
				for _, retained := range evidence.Retained {
					expected[retained.Unit.String()] = true
				}
			}
		}
	}
	retained := 0
	got := map[string]bool{}
	for _, pkg := range inspection.Inventory().Packages() {
		for _, file := range pkg.Files() {
			key := file.File().String()
			if !file.RootUnit().IsZero() {
				key = file.RootUnit().String()
			}
			if !expected[key] {
				t.Errorf("inventory row %s has no full-semantic evidence", key)
			}
			got[key] = true
			retained += len(file.Occurrences())
		}
	}
	for key := range expected {
		if !got[key] {
			t.Errorf("full-semantic evidence %s missing from the inventory", key)
		}
	}
	const retainedBudget = 2000 // calibrated: 469 retained; pre-partition class was 598,836
	if retained > retainedBudget {
		t.Errorf("retained occurrences %d exceed the reviewed budget %d — scope-class regression", retained, retainedBudget)
	}
	if retained == 0 {
		t.Error("no retained occurrences — full-semantic scope lost")
	}
}
