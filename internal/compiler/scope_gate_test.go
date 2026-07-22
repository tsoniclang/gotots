package compiler

import (
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// TestRetainedScopeExactJoin is the scope/cost gate's deterministic half: the
// region model exact-joins the finalized full-semantic unit set — every full
// unit owns exactly one body region and every body region roots at a full
// unit (both one-sided checks) — no declaration-contract unit contributes a
// region, and the retained-occurrence count sits inside the reviewed budget.
func TestRetainedScopeExactJoin(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "testdata", "projects", "webshop")
	inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	// Independent expected set: full-semantic units from the finalized census.
	full := map[string]bool{}
	for _, pkg := range inspection.Workspace().Packages() {
		for _, p := range pkg.Files() {
			for _, unit := range p.Units() {
				if unit.Depth() == source.DepthFullSemantic {
					full[unit.ID().String()] = true
				}
			}
		}
	}
	retained := 0
	bodyRegions := map[string]bool{}
	for _, pkg := range inspection.Inventory().Packages() {
		for _, region := range pkg.Files() {
			retained += len(region.Occurrences())
			if region.RootUnit().IsZero() {
				continue // file declaration region
			}
			key := region.RootUnit().String()
			if !full[key] {
				t.Errorf("body region %s has no full-semantic unit", key)
			}
			bodyRegions[key] = true
		}
	}
	for key := range full {
		if !bodyRegions[key] {
			t.Errorf("full-semantic unit %s has no body region", key)
		}
	}
	const retainedBudget = 3000 // webshop region model; a scope-class regression is orders larger
	if retained > retainedBudget {
		t.Errorf("retained occurrences %d exceed the reviewed budget %d — scope-class regression", retained, retainedBudget)
	}
	if retained == 0 {
		t.Error("no retained occurrences — full-semantic scope lost")
	}
}
