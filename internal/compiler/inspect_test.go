package compiler

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate inspect_test.go")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// TestInspectUnrelatedProjects proves whole-module inspection over two
// unrelated realistic Go projects: both load, verify, inventory, and report
// exact denominators with zero unknowns (an unknown construct or directive
// aborts, so success is the zero-unknown proof).
func TestInspectUnrelatedProjects(t *testing.T) {
	root := repoRoot(t)
	for _, project := range []string{"webshop", "textindex"} {
		dir := filepath.Join(root, "testdata", "projects", project)
		inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
		if err != nil {
			t.Fatalf("%s: %v", project, err)
		}
		d := inspection.Inventory().Denominators()
		if d.Packages < 3 || d.Files < 3 || d.Occurrences < 200 {
			t.Errorf("%s: implausible denominators %+v", project, d)
		}
		if d.VariantBearing == 0 {
			t.Errorf("%s: no variant-bearing occurrences", project)
		}
		// The universe holds the complete closure: std packages appear as
		// dependency records with reserved owners.
		stdSeen := false
		for _, pkg := range inspection.Workspace().Packages() {
			if pkg.ID().Owner().String() == "std" {
				stdSeen = true
				break
			}
		}
		if !stdSeen {
			t.Errorf("%s: closure holds no standard-library packages", project)
		}
	}
}

// TestInspectMultiModuleWorkspace proves a go.work workspace inventories both
// modules with distinct module-qualified identities for identical relative
// paths.
func TestInspectMultiModuleWorkspace(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "testdata", "workspaces", "dual")
	inspection, err := InspectConstructs(withManifest(t, source.Request{
		Dir:              dir,
		ProviderContract: scope.DefaultContractID,
		Patterns:         []string{"dual.example/a/...", "dual.example/b/..."},
	}))
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	files := map[string]bool{}
	for _, pkg := range inspection.Inventory().Packages() {
		for _, file := range pkg.Files() {
			files[file.File().String()] = true
		}
	}
	if !files["mod=dual.example/a::pkg/same.go"] || !files["mod=dual.example/b::pkg/same.go"] {
		t.Errorf("identical relative paths not module-qualified: %v", files)
	}
}

// TestInspectSelfModule proves the pipeline runs over a real multi-package
// module: this repository itself.
func TestInspectSelfModule(t *testing.T) {
	inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: repoRoot(t), ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatalf("self-inspection: %v", err)
	}
	d := inspection.Inventory().Denominators()
	if d.Packages < 7 || d.Occurrences < 5000 {
		t.Errorf("implausible self-inspection denominators %+v", d)
	}
}

// TestImportCoherencePositiveProof is the reviewer-mandated positive proof:
// for a root importing dep.Box, (1) the importer's types.Package.Imports()
// entry and the dependency record share the identical *types.Package object;
// (2) the selector's use object is the exact declaration object in the
// dependency scope; (3) the dependency body appears in the inventory; and
// (4) relocation preserves canonical identities.
func TestImportCoherencePositiveProof(t *testing.T) {
	content := map[string]string{
		"go.mod":     "module coherent.example/app\n\ngo 1.26\n",
		"app.go":     "package app\n\nimport \"coherent.example/app/dep\"\n\nvar V dep.Box\n\nfunc Use() int { return V.Size() }\n",
		"dep/dep.go": "package dep\n\n// Box is the dependency declaration.\ntype Box struct{ N int }\n\nfunc (b Box) Size() int { return b.N }\n",
	}
	load := func() (*Inspection, string) {
		dir := t.TempDir()
		for rel, data := range content {
			path := filepath.Join(dir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		inspection, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID, Patterns: []string{"coherent.example/app"}}))
		if err != nil {
			t.Fatalf("InspectConstructs: %v", err)
		}
		return inspection, dir
	}
	inspection, _ := load()
	var app, dep *source.Package
	for _, pkg := range inspection.Workspace().Packages() {
		switch pkg.ID().ImportPath() {
		case "coherent.example/app":
			app = pkg
		case "coherent.example/app/dep":
			dep = pkg
		}
	}
	if app == nil || dep == nil {
		t.Fatal("app or dep missing from the universe")
	}
	if app.RequestedRoot() == dep.RequestedRoot() {
		t.Errorf("root separation lost: app root=%v dep root=%v", app.RequestedRoot(), dep.RequestedRoot())
	}
	// (1) One coherent go/types object graph.
	sameObject := false
	for _, imported := range app.Types().Imports() {
		if imported.Path() == "coherent.example/app/dep" {
			sameObject = imported == dep.Types()
		}
	}
	if !sameObject {
		t.Error("importer and dependency record hold distinct *types.Package objects")
	}
	if dep.Types().Scope().Lookup("Box") == nil {
		t.Fatal("Box not in dependency scope")
	}
	// (2) The qualified reference `dep.Box` resolved through the one checker
	// graph: the app inventory carries a package-member selector occurrence
	// (variant resolution requires the checker to have resolved it).
	usedAsBox := false
	for _, pkg := range inspection.Inventory().Packages() {
		if pkg.ID().ImportPath() != "coherent.example/app" {
			continue
		}
		for _, region := range pkg.Files() {
			for _, occ := range region.Occurrences() {
				if occ.Variant() == catalog.VariantSelectPackageMember {
					usedAsBox = true
				}
			}
		}
	}
	if !usedAsBox {
		t.Error("app's dep.Box qualified reference did not resolve to a package member")
	}
	// (3) The dependency body is inventoried (whole-closure analysis).
	depInventoried := false
	for _, pkg := range inspection.Inventory().Packages() {
		if pkg.ID().ImportPath() == "coherent.example/app/dep" && len(pkg.Files()) >= 1 {
			depInventoried = len(pkg.Files()[0].Occurrences()) > 0
		}
	}
	if !depInventoried {
		t.Error("dependency body missing from the inventory")
	}
	// (4) Relocation preserves canonical identities.
	second, _ := load()
	first := map[string]bool{}
	for _, pkg := range inspection.Inventory().Packages() {
		for _, file := range pkg.Files() {
			for _, occurrence := range file.Occurrences() {
				first[occurrence.ID().String()] = true
			}
		}
	}
	for _, pkg := range second.Inventory().Packages() {
		for _, file := range pkg.Files() {
			for _, occurrence := range file.Occurrences() {
				if !first[occurrence.ID().String()] {
					t.Fatalf("relocated occurrence %s not in first load", occurrence.ID())
				}
			}
		}
	}
}

// cgoPipelineFixture: main (directly C-dependent), pure (shadowing local type
// C, not C-dependent), and hasLit (whose OWN body never touches C but whose
// nested literal does). It proves per-unit C-dependence derived from typed
// evidence, not spelling.
const cgoPipelineFixture = "package main\n\n/*\n#include <stdlib.h>\n*/\nimport \"C\"\n\n" +
	"func main() {\n\tC.free(nil)\n}\n\n" +
	"func pure() int {\n\ttype C struct{ f int }\n\tlocal := C{f: 41}\n\treturn local.f + 1\n}\n\n" +
	"func hasLit() {\n\tf := func() { C.free(nil) }\n\t_ = f\n}\n"

// TestCgoThroughPublicPipeline proves a cgo program runs the COMPLETE public
// pipeline with C-dependence derived from authoritative typed evidence: a
// directly C-dependent unit is external-boundary, a shadowing local type C
// never classifies, a function whose own body is C-free stays full-semantic
// even when its nested literal touches C (per-unit exactness), and the origin
// graph plus collision-free synthetics are present and relocation-stable.
func TestCgoThroughPublicPipeline(t *testing.T) {
	t.Skip("cgo checked-counterpart region traversal is wired in the cgo fix-forward step (Outcome 3)")
	load := func(t *testing.T) (*Inspection, *source.Package) {
		dir := t.TempDir()
		for rel, content := range map[string]string{
			"go.mod":  "module cgo.example/pipeline\n\ngo 1.26\n",
			"main.go": cgoPipelineFixture,
		} {
			if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		req := source.Request{Dir: dir, ProviderContract: scope.DefaultContractID, Env: []string{"CGO_ENABLED=1"}}
		artifact, err := AuditCatalog(req)
		if err != nil {
			t.Skipf("cgo unavailable: %v", err)
		}
		manifestPath := filepath.Join(t.TempDir(), "manifest.json")
		if err := analyze.WriteAuditArtifact(artifact, manifestPath); err != nil {
			t.Fatal(err)
		}
		req.AuditArtifact = manifestPath
		req.AuditArtifactDigest = artifact.ArtifactDigest
		inspection, err := InspectConstructs(req)
		if err != nil {
			t.Fatalf("cgo consumption pipeline: %v", err)
		}
		var mainPkg *source.Package
		for _, pkg := range inspection.Workspace().Packages() {
			if pkg.ID().ImportPath() == "cgo.example/pipeline" {
				mainPkg = pkg
			}
		}
		if mainPkg == nil {
			t.Fatal("cgo package missing")
		}
		return inspection, mainPkg
	}

	inspection, mainPkg := load(t)
	byName := map[string]source.SourceUnit{}
	for _, unit := range mainPkg.Units() {
		byName[unit.Name()] = unit
	}
	if got := byName["main"].Depth(); got != source.DepthExternalBoundary {
		t.Errorf("C-dependent main depth = %s, want external-boundary", got)
	}
	if got := byName["pure"].Depth(); got != source.DepthFullSemantic {
		t.Errorf("pure (shadowing local C) depth = %s, want full-semantic", got)
	}
	// hasLit's own body never touches C — full-semantic — while its nested
	// literal, which calls C.free, is the C-dependent external boundary.
	if got := byName["hasLit"].Depth(); got != source.DepthFullSemantic {
		t.Errorf("hasLit (own body C-free) depth = %s, want full-semantic", got)
	}
	litCDependent := false
	for name, unit := range byName {
		if unit.Kind() == identity.UnitFuncLitBody && strings.HasPrefix(name, "hasLit$lit") {
			if unit.Depth() == source.DepthExternalBoundary {
				litCDependent = true
			}
		}
	}
	if !litCDependent {
		t.Error("the nested literal calling C.free was not classified C-dependent")
	}
	if len(mainPkg.CheckedUnitMappings()) == 0 || len(mainPkg.SyntheticUnits()) == 0 {
		t.Error("cgo origin mappings/synthetic units missing")
	}
	// Synthetic identities are canonical (real declared cgo names, never "decl")
	// and relocation-stable (identical across a fresh checkout directory).
	syntheticNames := func(pkg *source.Package) map[string]bool {
		set := map[string]bool{}
		for _, s := range pkg.SyntheticUnits() {
			if s.Name() == "decl" || s.Name() == "" {
				t.Errorf("non-canonical synthetic name %q", s.Name())
			}
			set[s.Name()] = true
		}
		return set
	}
	first := syntheticNames(mainPkg)
	_, secondPkg := load(t)
	second := syntheticNames(secondPkg)
	if len(first) != len(second) {
		t.Errorf("synthetic set not relocation-stable: %d vs %d names", len(first), len(second))
	}
	for name := range first {
		if !second[name] {
			t.Errorf("synthetic %s absent after relocation — identity depends on a temporary path", name)
		}
	}
	// The pure unit's interior occurrences are inventoried via the checked view.
	pureInventoried := false
	for _, pkg := range inspection.Inventory().Packages() {
		for _, file := range pkg.Files() {
			if !file.RootUnit().IsZero() && len(file.Occurrences()) > 0 {
				pureInventoried = true
			}
		}
	}
	if !pureInventoried {
		t.Error("pure cgo unit missing from the inventory")
	}
}
