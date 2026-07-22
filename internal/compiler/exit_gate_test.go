package compiler

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestDeepNestedFunctionLiterals proves the pre-scope ledger is total over
// arbitrarily nested literals: Outer + literal + literal-in-literal +
// initializer literal = five independently selected units.
func TestDeepNestedFunctionLiterals(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod": "module deep.example/fl\n\ngo 1.26\n",
		"main.go": `package fl

var Init = func() int { return 7 }

func Outer() func() func() int {
	middle := func() func() int {
		inner := func() int { return 1 }
		return inner
	}
	return middle
}
`,
	})
	inspection, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	kinds := map[identity.UnitKind]int{}
	for _, pkg := range inspection.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, unit := range pkg.Units() {
			kinds[unit.Kind()]++
			if unit.Depth() != source.DepthFullSemantic {
				t.Errorf("unit %s depth = %s", unit.ID(), unit.Depth())
			}
		}
	}
	if kinds[identity.UnitFuncBody] != 1 || kinds[identity.UnitFuncLitBody] != 3 || kinds[identity.UnitVarInitializer] != 1 {
		t.Errorf("census = %v, want 1 func body + 3 literals + 1 initializer", kinds)
	}
}

// TestAuditDriftBattery proves the content-addressed audit and manifest
// reject every tamper class — zeroed counts, per-file byte drift under an
// overlay, duplicate rows, a correctly RESEALED manifest-unit omission, and
// build-context drift — while the untouched artifact both verifies as the
// gate and is consumed by ordinary compilation.
func TestAuditDriftBattery(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module drift.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nimport \"errors\"\n\nvar E = errors.New(\"x\")\n",
	})
	req := source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}
	artifact, err := AuditCatalog(req)
	if err != nil {
		t.Fatalf("AuditCatalog: %v", err)
	}
	write := func(a analyze.AuditArtifact) string {
		path := filepath.Join(t.TempDir(), "audit.json")
		if err := analyze.WriteAuditArtifact(&a, path); err != nil {
			t.Fatal(err)
		}
		return path
	}
	goodPath := write(*artifact)
	consume := func(path string) (*Inspection, error) {
		r := req
		r.AuditArtifact = path
		r.AuditArtifactDigest = artifact.ArtifactDigest
		return InspectConstructs(r)
	}
	inspection, err := consume(goodPath)
	if err != nil {
		t.Fatalf("consumption of the untouched artifact failed: %v", err)
	}
	consumeReq := req
	consumeReq.AuditArtifact = goodPath
	consumeReq.AuditArtifactDigest = artifact.ArtifactDigest
	if err := VerifyAuditArtifact(inspection, consumeReq, goodPath); err != nil {
		t.Fatalf("untouched artifact rejected: %v", err)
	}
	if err := AuditVerify(req, goodPath); err != nil {
		t.Fatalf("gate verification of the untouched artifact failed: %v", err)
	}
	// Seal battery: post-production tampering breaks the sealed digest.
	zeroed := *artifact
	zeroed.Files = append([]analyze.AuditFile(nil), artifact.Files...)
	for i := range zeroed.Files {
		zeroed.Files[i].Occurrences = 0
	}
	zeroed.Occurrences = 0
	if _, err := consume(write(zeroed)); err == nil {
		t.Error("zeroed counts consumed")
	}
	duplicated := *artifact
	duplicated.Files = append(append([]analyze.AuditFile(nil), artifact.Files...), artifact.Files[0])
	if _, err := consume(write(duplicated)); err == nil {
		t.Error("duplicate record consumed")
	}
	// Manifest-unit omission: the bounded local census cannot know the nested
	// unit existed, so authority is EXTERNAL — the request binds the certified
	// digest, and any content change (sealed or not) fails that binding on the
	// ordinary path.
	omitted := *artifact
	omitted.Files = append([]analyze.AuditFile(nil), artifact.Files...)
	dropped := false
	for i := range omitted.Files {
		if len(omitted.Files[i].Units) > 0 {
			omitted.Files[i].Units = append([]analyze.ManifestUnit(nil), omitted.Files[i].Units[1:]...)
			dropped = true
			break
		}
	}
	if !dropped {
		t.Fatal("no manifest unit available to drop")
	}
	if _, err := consume(write(omitted)); err == nil || !strings.Contains(err.Error(), "certified digest") {
		t.Errorf("manifest-unit omission not rejected by the external binding: %v", err)
	}
	// A CDependent flip is the same external-binding class.
	flipped := *artifact
	flipped.Files = append([]analyze.AuditFile(nil), artifact.Files...)
	flippedOne := false
	for i := range flipped.Files {
		if len(flipped.Files[i].Units) > 0 {
			units := append([]analyze.ManifestUnit(nil), flipped.Files[i].Units...)
			units[0].CDependent = !units[0].CDependent
			flipped.Files[i].Units = units
			flippedOne = true
			break
		}
	}
	if flippedOne {
		if _, err := consume(write(flipped)); err == nil || !strings.Contains(err.Error(), "certified digest") {
			t.Errorf("CDependent flip not rejected by the external binding: %v", err)
		}
	}
	// A validly sealed artifact under the WRONG certified digest is rejected.
	wrongDigest := req
	wrongDigest.AuditArtifact = goodPath
	wrongDigest.AuditArtifactDigest = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := InspectConstructs(wrongDigest); err == nil || !strings.Contains(err.Error(), "certified digest") {
		t.Errorf("valid artifact under the wrong certified digest consumed: %v", err)
	}
	// Overlay divergence: the same FileID selects different bytes than the
	// artifact recorded — consumption fails as stale, exactly.
	var stdFilePath string
	for _, pkg := range inspection.Workspace().Packages() {
		if pkg.ID().Owner().Class().String() == "standard-library" {
			for _, file := range pkg.Files() {
				stdFilePath = file.Path()
				break
			}
		}
		if stdFilePath != "" {
			break
		}
	}
	if stdFilePath == "" {
		t.Fatal("no std file in closure")
	}
	original, err := os.ReadFile(stdFilePath)
	if err != nil {
		t.Fatal(err)
	}
	overlaid := req
	overlaid.AuditArtifact = goodPath
	overlaid.AuditArtifactDigest = artifact.ArtifactDigest
	overlaid.Overlay = map[string][]byte{stdFilePath: append(append([]byte(nil), original...), []byte("\n// drift\n")...)}
	if _, err := InspectConstructs(overlaid); err == nil {
		t.Error("overlay whose selected bytes diverge from the manifest was consumed")
	}
	// Build-context drift: consumption under different build flags fails the
	// context join.
	driftedReq := req
	driftedReq.AuditArtifact = goodPath
	driftedReq.AuditArtifactDigest = artifact.ArtifactDigest
	driftedReq.BuildFlags = []string{"-tags=other"}
	if _, err := InspectConstructs(driftedReq); err == nil {
		t.Error("artifact consumed under drifted build context")
	}
}

// TestManifestSuppliedInteriors proves provider-owned nested function
// literals enter the selectable census only through the verified manifest —
// the bounded local census cannot produce them — and that every provider
// package carries its implicit initialization unit in the finalized ledger.
func TestManifestSuppliedInteriors(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module interiors.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nimport \"fmt\"\n\nfunc F() { fmt.Println(1) }\n",
	})
	inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	stdLits, stdImplicit := 0, 0
	for _, pkg := range inspection.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerStandardLibrary {
			continue
		}
		stdImplicit += len(pkg.ImplicitUnits())
		for _, unit := range pkg.Units() {
			if unit.Kind() == identity.UnitFuncLitBody {
				stdLits++
				if unit.Depth() == source.DepthInvalid {
					t.Errorf("manifest interior %s has no selected depth", unit.ID())
				}
			}
		}
	}
	if stdLits == 0 {
		t.Error("no manifest-supplied interior literals in the std closure")
	}
	if stdImplicit == 0 {
		t.Error("no implicit initialization units on provider packages")
	}
	// The same closure WITHOUT the artifact fails closed: provider files
	// require their manifest records.
	if _, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}); err == nil {
		t.Error("provider-owned closure inspected without a manifest")
	}
}

// TestFinalizedArtifactAccessorImmutability proves mutation attempts through
// every collection accessor leave the artifacts unchanged.
func TestFinalizedArtifactAccessorImmutability(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module immut.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nfunc F(a int) int { return a + 1 }\n",
	})
	inspection, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
	if err != nil {
		t.Fatal(err)
	}
	ws := inspection.Workspace()
	packagesBefore := len(ws.Packages())
	mutated := ws.Packages()
	mutated[0] = nil
	if got := ws.Packages(); len(got) != packagesBefore || got[0] == nil {
		t.Error("workspace package collection mutated through accessor")
	}
	pkg := ws.Roots()[0]
	filesBefore := len(pkg.Files())
	files := pkg.Files()
	files[0] = nil
	if got := pkg.Files(); len(got) != filesBefore || got[0] == nil {
		t.Error("package file collection mutated through accessor")
	}
	units := pkg.Units()
	if len(units) > 0 {
		units[0] = source.SourceUnit{}
		if pkg.Units()[0].ID().IsZero() {
			t.Error("unit collection mutated through accessor")
		}
	}
	imports := pkg.Imports()
	if len(imports) > 0 {
		imports[0] = "tampered"
		if pkg.Imports()[0] == "tampered" {
			t.Error("import collection mutated through accessor")
		}
	}
	inv := inspection.Inventory().Packages()[0].Files()[0]
	occurrencesBefore := len(inv.Occurrences())
	occurrences := inv.Occurrences()
	occurrences[0] = analyze.Occurrence{}
	if got := inv.Occurrences(); len(got) != occurrencesBefore || got[0].ID().IsZero() {
		t.Error("occurrence collection mutated through accessor")
	}
}

// nestedFixture is Outer with one nested literal — the reviewer's Correction-3
// fixture. Both units are censused; a custom contract splits them across depths
// to exercise exact per-unit retention.
const nestedFixture = `package fl

func Outer() {
	inner := func() {
		println("inner")
	}
	inner()
}
`

// mixedDepthContract writes a version-2 contract artifact: the default
// namespace rules plus one exact-unit rule binding target to gostdlib
// (declaration contract, i.e. non-full). Returns its path.
func mixedDepthContract(t *testing.T, id, targetUnit string) string {
	t.Helper()
	body := `{"id": "` + id + `", "version": 2, "rules": [
  {"bind": "namespace", "namespace": "module", "condition": "always", "provider": "automatic-translation"},
  {"bind": "namespace", "namespace": "standard-library", "condition": "always", "provider": "gostdlib"},
  {"bind": "namespace", "namespace": "toolchain", "condition": "always", "provider": "toolchain-source"},
  {"bind": "namespace", "namespace": "language-pseudo", "condition": "always", "provider": "language-intrinsic"},
  {"bind": "namespace", "namespace": "module", "condition": "bodyless", "provider": "external-obligation"},
  {"bind": "namespace", "namespace": "standard-library", "condition": "intrinsic-disposition", "provider": "language-intrinsic"},
  {"bind": "unit", "unit": "` + targetUnit + `", "condition": "always", "provider": "gostdlib"}
]}`
	path := filepath.Join(t.TempDir(), id+".json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// reachableRanges walks root and returns the physical offset range of every
// reachable node. When respect is true, descent halts at (and excludes) each
// boundary node — the structural exclusion; when false, boundaries are ignored.
func reachableRanges(root ast.Node, boundaries []ast.Node, respect bool, fset *token.FileSet) [][2]int {
	set := map[ast.Node]bool{}
	for _, b := range boundaries {
		set[b] = true
	}
	var out [][2]int
	ast.Inspect(root, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		if respect && set[n] {
			return false
		}
		out = append(out, [2]int{
			fset.PositionFor(n.Pos(), false).Offset,
			fset.PositionFor(n.End(), false).Offset,
		})
		return true
	})
	return out
}

// TestNestedMixedDepthExactRetention is the reviewer's Correction-3 proof: a
// custom contract splits Outer and its nested literal across depths in both
// directions; retention is exact per unit and no excluded body node remains
// reachable through the retained evidence.
func TestNestedMixedDepthExactRetention(t *testing.T) {
	dir := writeFixture(t, map[string]string{
		"go.mod":  "module fl.example/m\n\ngo 1.26\n",
		"main.go": nestedFixture,
	})
	base, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
	if err != nil {
		t.Fatalf("baseline inspect: %v", err)
	}
	var outer, inner source.SourceUnit
	for _, pkg := range base.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, unit := range pkg.Units() {
			switch unit.Kind() {
			case identity.UnitFuncBody:
				outer = unit
			case identity.UnitFuncLitBody:
				inner = unit
			}
		}
	}
	if outer.ID().IsZero() || inner.ID().IsZero() {
		t.Fatalf("did not census both units: outer=%v inner=%v", outer.ID(), inner.ID())
	}

	// resolve re-inspects under a contract making `nonFull` declaration-
	// contract and returns the single retained region (the full unit).
	resolve := func(t *testing.T, id string, nonFull, full source.SourceUnit) (*Inspection, source.RetainedUnit) {
		path := mixedDepthContract(t, id, nonFull.ID().String())
		insp, err := InspectConstructs(source.Request{
			Dir: dir, ProviderContract: id, ProviderContractArtifact: path,
		})
		if err != nil {
			t.Fatalf("mixed-depth inspect: %v", err)
		}
		var regions []source.RetainedUnit
		depth := map[string]source.EvidenceDepth{}
		for _, pkg := range insp.Workspace().Packages() {
			if pkg.ID().Owner().Class() != identity.OwnerModule {
				continue
			}
			for _, unit := range pkg.Units() {
				depth[unit.ID().String()] = unit.Depth()
			}
			for _, file := range pkg.Files() {
				if mixed, ok := file.Evidence().(source.MixedUnits); ok {
					regions = append(regions, mixed.Retained...)
				}
			}
		}
		if depth[nonFull.ID().String()] != source.DepthDeclarationContract {
			t.Errorf("non-full unit %s depth = %s, want declaration-contract", nonFull.ID(), depth[nonFull.ID().String()])
		}
		if depth[full.ID().String()] != source.DepthFullSemantic {
			t.Errorf("full unit %s depth = %s, want full-semantic", full.ID(), depth[full.ID().String()])
		}
		if len(regions) != 1 || regions[0].Unit != full.ID() {
			t.Fatalf("retained regions = %v, want exactly the full unit %s", regions, full.ID())
		}
		return insp, regions[0]
	}

	// Case B: Outer full, inner non-full. The excluded inner body must be
	// unreachable through Outer's retained region — structurally, via the
	// boundary — while a boundary-ignoring walk still reaches it (load-bearing).
	t.Run("outer-full-inner-nonfull", func(t *testing.T) {
		insp, region := resolve(t, "caseb@v1", inner, outer)
		fset := insp.Workspace().Fset()
		innerSpan := inner.Span()
		strictlyInsideInner := func(ranges [][2]int) int {
			n := 0
			for _, r := range ranges {
				if r[0] >= innerSpan.Start.Offset && r[1] <= innerSpan.End.Offset &&
					!(r[0] == innerSpan.Start.Offset && r[1] == innerSpan.End.Offset) {
					n++
				}
			}
			return n
		}
		if got := strictlyInsideInner(reachableRanges(region.Decl, region.Boundaries, true, fset)); got != 0 {
			t.Errorf("%d excluded nested-body nodes reachable through the retained region", got)
		}
		if got := strictlyInsideInner(reachableRanges(region.Decl, region.Boundaries, false, fset)); got == 0 {
			t.Error("boundary-ignoring walk reached no nested body — the exclusion proof is vacuous")
		}
	})

	// Case A: inner full, Outer non-full. Retaining inner must not drag in
	// Outer's body: the region roots at the FuncLit (span == inner), and every
	// reachable node lies within inner's span — no Outer-only node.
	t.Run("inner-full-outer-nonfull", func(t *testing.T) {
		insp, region := resolve(t, "casea@v1", outer, inner)
		fset := insp.Workspace().Fset()
		start := fset.PositionFor(region.Decl.Pos(), false).Offset
		end := fset.PositionFor(region.Decl.End(), false).Offset
		if start != inner.Span().Start.Offset || end != inner.Span().End.Offset {
			t.Errorf("retained root span %d-%d, want the nested literal span %d-%d (not the enclosing Outer)",
				start, end, inner.Span().Start.Offset, inner.Span().End.Offset)
		}
		outerOnly := 0
		for _, r := range reachableRanges(region.Decl, region.Boundaries, true, fset) {
			insideOuter := r[0] >= outer.Span().Start.Offset && r[1] <= outer.Span().End.Offset
			insideInner := r[0] >= inner.Span().Start.Offset && r[1] <= inner.Span().End.Offset
			if insideOuter && !insideInner {
				outerOnly++
			}
		}
		if outerOnly != 0 {
			t.Errorf("%d enclosing-Outer nodes reachable through the retained nested unit", outerOnly)
		}
	})
}
