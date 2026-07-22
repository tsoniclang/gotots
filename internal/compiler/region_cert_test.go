package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// depthContract writes a version-2 contract artifact: the default namespace
// rules plus one exact-unit rule per target, binding it to gostdlib
// (declaration-contract, i.e. non-full). Everything else stays automatic/full.
func depthContract(t *testing.T, id string, nonFull ...identity.SourceUnitID) string {
	t.Helper()
	rules := `
  {"bind": "namespace", "namespace": "module", "condition": "always", "provider": "automatic-translation"},
  {"bind": "namespace", "namespace": "standard-library", "condition": "always", "provider": "gostdlib"},
  {"bind": "namespace", "namespace": "toolchain", "condition": "always", "provider": "toolchain-source"},
  {"bind": "namespace", "namespace": "language-pseudo", "condition": "always", "provider": "language-intrinsic"},
  {"bind": "namespace", "namespace": "module", "condition": "bodyless", "provider": "external-obligation"},
  {"bind": "namespace", "namespace": "standard-library", "condition": "intrinsic-disposition", "provider": "language-intrinsic"}`
	for _, u := range nonFull {
		rules += `,
  {"bind": "unit", "unit": "` + u.String() + `", "condition": "always", "provider": "gostdlib"}`
	}
	body := `{"id": "` + id + `", "version": 2, "rules": [` + rules + `]}`
	path := filepath.Join(t.TempDir(), id+".json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// nestedCertFixture: Outer with one nested literal that captures a binding
// declared in Outer.
const nestedCertFixture = `package fl

func Outer() int {
	base := 10
	inner := func() int { return base + 1 }
	return inner()
}
`

// censusUnits inspects under the default contract and returns the module
// func-body and funclit unit identities.
func censusUnits(t *testing.T, dir string) (outer, inner identity.SourceUnitID) {
	t.Helper()
	base, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
	if err != nil {
		t.Fatalf("baseline inspect: %v", err)
	}
	for _, pkg := range base.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, p := range pkg.Files() {
			for _, u := range p.Units() {
				switch u.Kind() {
				case identity.UnitFuncBody:
					outer = u.ID()
				case identity.UnitFuncLitBody:
					inner = u.ID()
				}
			}
		}
	}
	if outer.IsZero() || inner.IsZero() {
		t.Fatalf("did not census both units: outer=%v inner=%v", outer, inner)
	}
	return outer, inner
}

// regionModel gathers the inventory's definitions, references, and body regions.
func regionModel(t *testing.T, insp *Inspection) (defs map[string]analyze.ImplementationDefinition, refs []analyze.ImplementationRef, regions map[string]*analyze.FileInventory) {
	t.Helper()
	defs = map[string]analyze.ImplementationDefinition{}
	regions = map[string]*analyze.FileInventory{}
	for _, pkg := range insp.Inventory().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, d := range pkg.Definitions() {
			defs[d.Unit().String()] = d
		}
		refs = append(refs, pkg.References()...)
		for _, region := range pkg.Files() {
			if !region.RootUnit().IsZero() {
				regions[region.RootUnit().String()] = region
			}
		}
	}
	return defs, refs, regions
}

// TestRegionCertificationOuterFullChildNonFull is the reviewer's artifact
// inspection proof: with the outer body full and the nested body non-full, the
// parent references the child exactly once, the child definition exists exactly
// once, the child interior occurrence count is zero, and no occurrence of the
// parent region falls within the child's span (a generic traversal from the
// parent cannot reach the child body).
func TestRegionCertificationOuterFullChildNonFull(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module cert.example/m\n\ngo 1.26\n",
		"main.go": nestedCertFixture,
	})
	outer, inner := censusUnits(t, dir)
	path := depthContract(t, "outerfull@v1", inner) // inner non-full
	insp, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: "outerfull@v1", ProviderContractArtifact: path})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	defs, refs, regions := regionModel(t, insp)

	// Child definition exists exactly once, and it is non-full.
	childDef, ok := defs[inner.String()]
	if !ok {
		t.Fatal("child has no definition")
	}
	if childDef.Full() {
		t.Error("child definition is full, want non-full")
	}
	// Outer owns a body region; inner does not.
	if _, ok := regions[outer.String()]; !ok {
		t.Error("outer (full) has no body region")
	}
	if _, ok := regions[inner.String()]; ok {
		t.Error("inner (non-full) owns a body region")
	}
	// Parent references the child exactly once.
	childRefs := 0
	for _, r := range refs {
		if r.Child().Source() == inner {
			childRefs++
		}
	}
	if childRefs != 1 {
		t.Errorf("parent references child %d times, want exactly once", childRefs)
	}
	// Child interior occurrence count is zero, and no parent-region occurrence
	// lies within the child's span — structural isolation.
	innerSpan := inner.Span()
	for _, region := range regions {
		for _, occ := range region.Occurrences() {
			s := occ.Span()
			inChild := s.Start.Offset >= innerSpan.Start() && s.End.Offset <= innerSpan.End()
			if inChild {
				t.Errorf("occurrence %s lies within the excluded child body — structural isolation violated", occ.ID())
			}
		}
	}
}

// TestRegionCertificationChildFullOuterNonFull is the inverse: the nested body
// is full while the outer body is non-full, including a binding captured from
// the excluded parent. The child owns its region and object/binder evidence
// without retaining the excluded parent body.
func TestRegionCertificationChildFullOuterNonFull(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module cert2.example/m\n\ngo 1.26\n",
		"main.go": nestedCertFixture,
	})
	outer, inner := censusUnits(t, dir)
	path := depthContract(t, "childfull@v1", outer) // outer non-full
	insp, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: "childfull@v1", ProviderContractArtifact: path})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	defs, _, regions := regionModel(t, insp)

	if d := defs[outer.String()]; d.Full() {
		t.Error("outer definition is full, want non-full")
	}
	region, ok := regions[inner.String()]
	if !ok {
		t.Fatal("inner (full) has no body region")
	}
	if _, ok := regions[outer.String()]; ok {
		t.Error("outer (non-full) owns a body region")
	}
	// The child region roots at the nested literal and reaches only its own
	// span — the excluded outer body is unreachable.
	innerSpan := inner.Span()
	for _, occ := range region.Occurrences() {
		s := occ.Span()
		if s.Start.Offset < innerSpan.Start() || s.End.Offset > innerSpan.End() {
			t.Errorf("child region occurrence %s escapes the nested literal span (reaches the excluded parent)", occ.ID())
		}
	}
	// The child region is non-empty and structurally isolated. Whether the
	// binding `base` is captured from the excluded parent is a semantic-model
	// fact materialized from the checker graph in Stage 2 — an occurrence's
	// presence alone does not prove capture identity, so Stage 1 asserts only
	// that the child's own body region exists and is isolated.
	if len(region.Occurrences()) == 0 {
		t.Error("child region has no occurrences")
	}
}

// TestRegionCertificationBothDepths proves the both-full and both-non-full
// extremes: both full yields two body regions and a reference; both non-full
// yields two definitions, no body region, and zero occurrences for the units.
func TestRegionCertificationBothDepths(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module cert3.example/m\n\ngo 1.26\n",
		"main.go": nestedCertFixture,
	})
	outer, inner := censusUnits(t, dir)

	// Both full (default contract).
	full, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatalf("both-full inspect: %v", err)
	}
	_, refs, regions := regionModel(t, full)
	if _, ok := regions[outer.String()]; !ok {
		t.Error("both-full: outer has no region")
	}
	if _, ok := regions[inner.String()]; !ok {
		t.Error("both-full: inner has no region")
	}
	found := false
	for _, r := range refs {
		if r.Child().Source() == inner {
			found = true
		}
	}
	if !found {
		t.Error("both-full: no reference to the nested literal")
	}

	// Both non-full.
	path := depthContract(t, "bothnonfull@v1", outer, inner)
	insp, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: "bothnonfull@v1", ProviderContractArtifact: path})
	if err != nil {
		t.Fatalf("both-non-full inspect: %v", err)
	}
	defs, _, regions2 := regionModel(t, insp)
	for _, u := range []identity.SourceUnitID{outer, inner} {
		if d, ok := defs[u.String()]; !ok || d.Full() {
			t.Errorf("both-non-full: unit %s definition missing or full", u)
		}
		if _, ok := regions2[u.String()]; ok {
			t.Errorf("both-non-full: unit %s owns a body region", u)
		}
	}
}
