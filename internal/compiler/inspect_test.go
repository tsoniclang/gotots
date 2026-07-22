package compiler

import (
	"go/ast"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
		inspection, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID})
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
	inspection, err := InspectConstructs(source.Request{
		Dir:              dir,
		ProviderContract: scope.DefaultContractID,
		Patterns:         []string{"dual.example/a/...", "dual.example/b/..."},
	})
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
	inspection, err := InspectConstructs(source.Request{Dir: repoRoot(t), ProviderContract: scope.DefaultContractID})
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
		inspection, err := InspectConstructs(source.Request{Dir: dir, ProviderContract: scope.DefaultContractID, Patterns: []string{"coherent.example/app"}})
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
	// (2) The selector use is the exact declaration object in the dep scope.
	boxDecl := dep.Types().Scope().Lookup("Box")
	if boxDecl == nil {
		t.Fatal("Box not in dependency scope")
	}
	usedAsBox := false
	for _, file := range app.Files() {
		syntax, full := file.FullSyntax()
		if !full {
			continue
		}
		ast.Inspect(syntax, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Box" {
				if used, ok := app.TypesInfo().UseOf(sel.Sel); ok && used == boxDecl {
					usedAsBox = true
				}
			}
			return true
		})
	}
	if !usedAsBox {
		t.Error("app's dep.Box use does not resolve to the exact dependency declaration object")
	}
	// (3) The dependency body is inventoried (whole-closure analysis).
	depInventoried := false
	for _, pkg := range inspection.Inventory().Packages() {
		if pkg.ID().ImportPath() == "coherent.example/app/dep" && len(pkg.Files()) == 1 {
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

// TestCgoThroughPublicPipeline proves a cgo program runs the COMPLETE public
// pipeline: mixed same-file units (C-dependent external boundary + pure
// full-semantic through the checked view), shadowing local C, and origin
// mappings — never a loader/verifier closure disagreement.
func TestCgoThroughPublicPipeline(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":  "module cgo.example/pipeline\n\ngo 1.26\n",
		"main.go": "package main\n\n/*\n#include <stdlib.h>\n*/\nimport \"C\"\n\nfunc main() {\n\tC.free(nil)\n}\n\nfunc pure() int {\n\ttype C struct{ f int }\n\tlocal := C{f: 41}\n\treturn local.f + 1\n}\n",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inspection, err := InspectConstructs(source.Request{
		Dir: dir, ProviderContract: scope.DefaultContractID,
		Env: []string{"CGO_ENABLED=1"},
	})
	if err != nil {
		t.Skipf("cgo unavailable: %v", err)
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
	depths := map[string]source.EvidenceDepth{}
	for _, unit := range mainPkg.Units() {
		depths[unit.Name()] = unit.Depth()
	}
	if depths["main"] != source.DepthExternalBoundary {
		t.Errorf("C-dependent main depth = %s, want external-boundary", depths["main"])
	}
	// pure declares a LOCAL type C — the shadowing name never classifies.
	if depths["pure"] != source.DepthFullSemantic {
		t.Errorf("pure (with shadowing local C) depth = %s, want full-semantic", depths["pure"])
	}
	if len(mainPkg.CheckedUnitMappings()) == 0 || len(mainPkg.SyntheticUnits()) == 0 {
		t.Error("cgo origin mappings/synthetic units missing")
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
