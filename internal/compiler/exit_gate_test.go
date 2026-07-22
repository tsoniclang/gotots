package compiler

import (
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
