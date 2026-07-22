package source_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// finalizeLoad runs the full source pipeline: transient load, default-contract
// scope selection, finalization.
func finalizeLoad(req source.Request) (*source.Workspace, error) {
	universe, err := source.LoadUniverse(req)
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, scope.DefaultContract())
	if err != nil {
		return nil, err
	}
	return source.Finalize(universe, selection.Depths())
}

// writeTree writes a file tree under dir.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func findPackage(t *testing.T, ws *source.Workspace, importPath string) *source.Package {
	t.Helper()
	for _, pkg := range ws.Packages() {
		if pkg.ID().ImportPath() == importPath {
			return pkg
		}
	}
	t.Fatalf("package %s not in closure", importPath)
	return nil
}

// TestLoadWorkspaceSingleModule proves a module loads into typed selected
// packages with owner-qualified machine-independent file identities.
func TestLoadWorkspaceSingleModule(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":      "module example.com/single\n\ngo 1.26\n",
		"main.go":     "package main\n\nfunc main() { _ = add(1, 2) }\n\nfunc add(a, b int) int { return a + b }\n",
		"pkg/util.go": "package pkg\n\nfunc Twice(x int) int { return 2 * x }\n",
	})
	ws, err := finalizeLoad(source.Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if len(ws.Roots()) != 2 {
		t.Fatalf("selected %d packages, want 2", len(ws.Roots()))
	}
	if ws.Toolchain().Binary() == "" || ws.Toolchain().Version() == "" || ws.Toolchain().GOROOT() == "" {
		t.Errorf("toolchain not resolved: %+v", ws.Toolchain())
	}
	var ids []string
	for _, pkg := range ws.Roots() {
		if pkg.Provenance() != source.ProvenanceWorkspaceModule || pkg.Acquisition() != source.AcquisitionWorkspace {
			t.Errorf("%s provenance/acquisition = %s/%s", pkg.ID(), pkg.Provenance(), pkg.Acquisition())
		}
		if pkg.ModuleGoVersion() != "1.26" {
			t.Errorf("%s module go directive = %q", pkg.ID(), pkg.ModuleGoVersion())
		}
		for _, file := range pkg.Files() {
			ids = append(ids, file.ID().String())
			if _, full := file.FullSyntax(); !full {
				t.Errorf("%s: workspace-module file lacks full-syntax evidence", file.ID())
			}
			if file.EffectiveGoVersion() == "" {
				t.Errorf("%s: no effective language version", file.ID())
			}
		}
	}
	want := []string{"mod=example.com/single::main.go", "mod=example.com/single::pkg/util.go"}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("file identity %d = %q, want %q", i, ids[i], want[i])
		}
	}
}

// TestUniverseClosureAndProvenance proves the complete transitive closure is
// resolved with correct owner/provenance/acquisition/disposition facts:
// standard library (fmt via import), unsafe intrinsic, a cache-acquired
// versioned dependency, and a locally replaced dotless module dependency.
func TestUniverseClosureAndProvenance(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":          "module universe.example/app\n\ngo 1.26\n\nrequire (\n\tdotlessdep/lib v1.0.0\n\tgolang.org/x/sync v0.21.0\n)\n\nreplace dotlessdep/lib => ./localdep\n",
		"go.sum":          "golang.org/x/sync v0.21.0 h1:HLII4xRRTtCRkxYp4HNFF0Js/Og6q2i++KXbg0gHCwM=\ngolang.org/x/sync v0.21.0/go.mod h1:9xrNwdLfx4jkKbNva9FpL6vEN7evnE43NNNJQ2LF3+0=\n",
		"main.go":         "package main\n\nimport (\n\t\"fmt\"\n\t\"unsafe\"\n\n\t\"dotlessdep/lib\"\n\t\"golang.org/x/sync/errgroup\"\n)\n\nfunc main() {\n\tvar g errgroup.Group\n\t_ = g.Wait()\n\tfmt.Println(lib.Answer(), unsafe.Sizeof(0))\n}\n",
		"localdep/go.mod": "module dotlessdep/lib\n\ngo 1.24\n",
		"localdep/lib.go": "package lib\n\nfunc Answer() int { return 42 }\n",
	})
	req := source.Request{Dir: dir, Env: []string{"GOFLAGS=-mod=mod", "GOPROXY=off"}}
	ws, err := finalizeLoad(req)
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	fmtPkg := findPackage(t, ws, "fmt")
	if fmtPkg.ID().Owner().String() != "std" || fmtPkg.Provenance() != source.ProvenanceStandardLibrary ||
		fmtPkg.Acquisition() != source.AcquisitionGOROOT || fmtPkg.RequestedRoot() {
		t.Errorf("fmt: %s %s %s root=%v", fmtPkg.ID(), fmtPkg.Provenance(), fmtPkg.Acquisition(), fmtPkg.RequestedRoot())
	}
	unsafePkg := findPackage(t, ws, "unsafe")
	if unsafePkg.Disposition() != source.DispositionUnsafeIntrinsic || unsafePkg.Provenance() != source.ProvenanceStandardLibrary {
		t.Errorf("unsafe: disposition=%s provenance=%s", unsafePkg.Disposition(), unsafePkg.Provenance())
	}
	sync := findPackage(t, ws, "golang.org/x/sync/errgroup")
	if sync.ID().Owner().String() != "mod=golang.org/x/sync@v0.21.0" {
		t.Errorf("x/sync owner = %s", sync.ID().Owner())
	}
	if sync.Provenance() != source.ProvenanceModuleDependency || sync.Acquisition() != source.AcquisitionModuleCache {
		t.Errorf("x/sync provenance/acquisition = %s/%s", sync.Provenance(), sync.Acquisition())
	}
	dotless := findPackage(t, ws, "dotlessdep/lib")
	if dotless.Provenance() != source.ProvenanceModuleDependency || dotless.Acquisition() != source.AcquisitionLocalReplacement {
		t.Errorf("dotless replacement provenance/acquisition = %s/%s (spelling must not classify)",
			dotless.Provenance(), dotless.Acquisition())
	}
	// A local replacement keeps the declared module path AND required
	// version; only acquisition reflects the replacement directory.
	if dotless.ID().Owner().String() != "mod=dotlessdep/lib@v1.0.0" {
		t.Errorf("dotless owner = %s", dotless.ID().Owner())
	}
	// Std leakage is structurally impossible: no body-indexed type
	// information survives finalization for declaration-contract packages.
	if fmtPkg.TypesInfo() != nil {
		t.Error("std package retains body-indexed type information")
	}
	// x/sync is a source-available module dependency: automatic provider,
	// full-semantic units retained.
	for _, unit := range sync.Units() {
		if unit.Depth() != source.DepthFullSemantic {
			t.Errorf("module-dependency unit %s depth = %s, want full-semantic", unit.ID(), unit.Depth())
		}
	}
	// Dependency records retain declaration-level type evidence through the
	// same loader (export data): fmt's Sprintln is visible without fmt being
	// selected.
	if fmtPkg.Types() == nil || fmtPkg.Types().Scope().Lookup("Sprintln") == nil {
		t.Error("fmt dependency record lacks declaration type evidence")
	}
	if sync.Types() == nil || sync.Types().Scope().Lookup("Group") == nil {
		t.Error("x/sync dependency record lacks declaration type evidence")
	}
	// Every closure record carries declaration-level type evidence; the
	// builtin pseudo-package is the single legitimate exception.
	var typeless []string
	for _, pkg := range ws.Packages() {
		if pkg.Types() == nil {
			typeless = append(typeless, pkg.ID().String())
		}
	}
	if len(typeless) != 0 {
		t.Errorf("closure records without type evidence: %v", typeless)
	}
	// Deep std dependencies resolve declarations without being selected.
	if cmpPkg := findPackage(t, ws, "errors"); cmpPkg.Types().Scope().Lookup("New") == nil {
		t.Error("errors dependency record lacks declaration type evidence")
	}
	// Std file identities are GOROOT/src-relative, never machine paths.
	for _, file := range fmtPkg.Files() {
		if !strings.HasPrefix(file.ID().String(), "std::fmt/") {
			t.Errorf("std file identity = %s", file.ID())
		}
		if strings.Contains(file.ID().String(), ws.Toolchain().GOROOT()) {
			t.Errorf("std identity embeds GOROOT: %s", file.ID())
		}
	}
}

// TestVendoredDependency proves a vendored dotless module is a module
// dependency with vendor acquisition and module-relative identity.
func TestVendoredDependency(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":                       "module vend.example/app\n\ngo 1.26\n\nrequire dotlessdep/lib v1.0.0\n",
		"main.go":                      "package main\n\nimport \"dotlessdep/lib\"\n\nfunc main() { _ = lib.Answer() }\n",
		"vendor/modules.txt":           "# dotlessdep/lib v1.0.0\n## explicit; go 1.24\ndotlessdep/lib\n",
		"vendor/dotlessdep/lib/lib.go": "package lib\n\nfunc Answer() int { return 42 }\n",
	})
	ws, err := finalizeLoad(source.Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	dotless := findPackage(t, ws, "dotlessdep/lib")
	if dotless.Provenance() != source.ProvenanceModuleDependency || dotless.Acquisition() != source.AcquisitionVendor {
		t.Errorf("vendored dep provenance/acquisition = %s/%s", dotless.Provenance(), dotless.Acquisition())
	}
	if got := dotless.Files()[0].ID().String(); got != "mod=dotlessdep/lib@v1.0.0::lib.go" {
		t.Errorf("vendored file identity = %s", got)
	}
}

// TestStdBuiltinAndToolchainRoots proves std, builtin, and toolchain command
// packages load as roots with reserved owners — Module==nil is never a
// rejection rule.
func TestStdBuiltinAndToolchainRoots(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"go.mod": "module roots.example/m\n\ngo 1.26\n", "m.go": "package m\n"})
	ws, err := finalizeLoad(source.Request{Dir: dir, Patterns: []string{"fmt", "builtin", "cmd/addr2line"}})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	fmtPkg := findPackage(t, ws, "fmt")
	if !fmtPkg.RequestedRoot() || fmtPkg.Types() == nil || len(fmtPkg.Files()) == 0 {
		t.Error("fmt root did not load with types and files")
	}
	// Under the default provider contract, std bodies are gostdlib-owned:
	// declaration-contract evidence, no retained syntax, boundaries present.
	for _, file := range fmtPkg.Files() {
		if _, full := file.FullSyntax(); full {
			t.Errorf("std file %s retains full syntax under the default contract", file.ID())
		}
	}
	stdUnits := fmtPkg.Units()
	if len(stdUnits) == 0 {
		t.Error("std package has no censused unit boundaries")
	}
	for _, unit := range stdUnits {
		if unit.Depth() != source.DepthDeclarationContract {
			t.Errorf("std unit %s depth = %s, want declaration-contract", unit.ID(), unit.Depth())
		}
		if unit.Hash().IsZero() {
			t.Errorf("std unit %s lacks a source-span hash", unit.ID())
		}
	}
	if fmtPkg.ID().String() != "std::fmt" {
		t.Errorf("fmt id = %s", fmtPkg.ID())
	}
	builtin := findPackage(t, ws, "builtin")
	if builtin.ID().Owner().String() != "lang" || builtin.Disposition() != source.DispositionBuiltinUniverse {
		t.Errorf("builtin owner/disposition = %s/%s", builtin.ID().Owner(), builtin.Disposition())
	}
	tool := findPackage(t, ws, "cmd/addr2line")
	if tool.ID().Owner().String() != "toolchain" || tool.Provenance() != source.ProvenanceToolchainPackage {
		t.Errorf("cmd/addr2line owner/provenance = %s/%s", tool.ID().Owner(), tool.Provenance())
	}
	var typeless []string
	for _, pkg := range ws.Packages() {
		if pkg.Types() == nil && pkg.ID().Owner().String() != "lang" {
			typeless = append(typeless, pkg.ID().String())
		}
	}
	if len(typeless) != 0 {
		t.Errorf("closure records without type evidence (only lang::builtin may lack it): %v", typeless)
	}
}

// TestPerFileEffectiveVersions proves the effective language version is a
// per-file fact from typed toolchain evidence: two files of one module differ
// when a //go:build constraint raises one file's version. There is no
// workspace maximum.
func TestPerFileEffectiveVersions(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.work":      "go 1.26\n\nuse (\n\t./old\n\t./new\n)\n",
		"old/go.mod":   "module ver.example/old\n\ngo 1.21\n",
		"old/base.go":  "package old\n\nfunc Base() int { return 1 }\n",
		"old/newer.go": "//go:build go1.26\n\npackage old\n\nfunc Newer() int { return 2 }\n",
		"new/go.mod":   "module ver.example/new\n\ngo 1.26\n",
		"new/n.go":     "package new\n\nfunc N() int { return 3 }\n",
	})
	ws, err := finalizeLoad(source.Request{Dir: dir, Patterns: []string{"ver.example/old/...", "ver.example/new/..."}})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	versions := map[string]string{}
	moduleDirectives := map[string]string{}
	for _, pkg := range ws.Roots() {
		moduleDirectives[pkg.ID().Owner().Module().Path()] = pkg.ModuleGoVersion()
		for _, file := range pkg.Files() {
			versions[file.ID().Rel()] = file.EffectiveGoVersion()
		}
	}
	if moduleDirectives["ver.example/old"] != "1.21" || moduleDirectives["ver.example/new"] != "1.26" {
		t.Errorf("module directives = %v", moduleDirectives)
	}
	if versions["base.go"] != "go1.21" {
		t.Errorf("base.go effective version = %q, want go1.21", versions["base.go"])
	}
	if versions["newer.go"] != "go1.26" {
		t.Errorf("newer.go effective version = %q, want go1.26 (per-file override)", versions["newer.go"])
	}
	if versions["n.go"] != "go1.26" {
		t.Errorf("n.go effective version = %q", versions["n.go"])
	}
}

// TestCgoViewDifferenceIsModeled proves a cgo package does not abort the
// loader: the source files keep owner identity, the checked-view difference
// is recorded explicitly, and per-body obligations remain for later planning.
func TestCgoViewDifferenceIsModeled(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":  "module cgo.example/m\n\ngo 1.26\n",
		"main.go": "package main\n\n/*\n#include <stdlib.h>\n*/\nimport \"C\"\n\nfunc main() { C.free(nil) }\n",
		"pure.go": "package main\n\nfunc pure() int { return 1 }\n",
	})
	ws, err := finalizeLoad(source.Request{Dir: dir, Env: []string{"CGO_ENABLED=1"}})
	if err != nil {
		t.Skipf("cgo unavailable in this environment: %v", err)
	}
	main := findPackage(t, ws, "cgo.example/m")
	// The checked view is modeled through origin-joined mappings and typed
	// synthetic units — never raw checked paths.
	if len(main.CheckedUnitMappings()) == 0 {
		t.Error("cgo package records no origin-joined checked-unit mappings")
	}
	if len(main.SyntheticUnits()) == 0 {
		t.Error("cgo package records no typed synthetic checked units")
	}
	for _, unit := range main.Units() {
		if unit.CDependent() && unit.Depth() != source.DepthExternalBoundary {
			t.Errorf("C-dependent unit %s depth = %s, want external-boundary", unit.ID(), unit.Depth())
		}
		if !unit.CDependent() && unit.Depth() != source.DepthFullSemantic {
			t.Errorf("pure unit %s depth = %s, want full-semantic", unit.ID(), unit.Depth())
		}
	}
	ids := map[string]bool{}
	for _, file := range main.Files() {
		ids[file.ID().String()] = true
	}
	if !ids["mod=cgo.example/m::main.go"] || !ids["mod=cgo.example/m::pure.go"] {
		t.Errorf("cgo source files lost owner identity: %v", ids)
	}
}

// TestRelocatedWorkspaceAndCache proves identity survives relocation of both
// the workspace checkout and the module cache: acquisition changes, identity
// does not.
func TestRelocatedWorkspaceAndCache(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	syncCache := filepath.Join(home, "go", "pkg", "mod", "cache", "download", "golang.org", "x", "sync")
	if _, err := os.Stat(syncCache); err != nil {
		t.Skip("x/sync not in the ambient module cache")
	}
	content := map[string]string{
		"go.mod": "module reloc.example/app\n\ngo 1.26\n\nrequire golang.org/x/sync v0.21.0\n",
		"go.sum": "golang.org/x/sync v0.21.0 h1:HLII4xRRTtCRkxYp4HNFF0Js/Og6q2i++KXbg0gHCwM=\ngolang.org/x/sync v0.21.0/go.mod h1:9xrNwdLfx4jkKbNva9FpL6vEN7evnE43NNNJQ2LF3+0=\n",
		"m.go":   "package main\n\nimport \"golang.org/x/sync/errgroup\"\n\nfunc main() { var g errgroup.Group; _ = g.Wait() }\n",
	}
	load := func(env []string) []string {
		dir := t.TempDir()
		writeTree(t, dir, content)
		ws, err := finalizeLoad(source.Request{Dir: dir, Env: env})
		if err != nil {
			t.Fatalf("LoadWorkspace: %v", err)
		}
		var ids []string
		for _, pkg := range ws.Packages() {
			ids = append(ids, pkg.ID().String())
		}
		return ids
	}
	baseline := load([]string{"GOFLAGS=-mod=mod", "GOPROXY=off"})
	// Relocate the module cache: copy the download tree into a fresh
	// GOMODCACHE and load from there.
	relocatedCache := t.TempDir()
	if err := os.CopyFS(filepath.Join(relocatedCache, "cache", "download", "golang.org", "x", "sync"), os.DirFS(syncCache)); err != nil {
		t.Fatalf("cache relocation: %v", err)
	}
	t.Cleanup(func() {
		// go extracts modules read-only; restore write permission so the
		// TempDir cleanup can remove the relocated cache.
		_ = filepath.WalkDir(relocatedCache, func(path string, d os.DirEntry, err error) error {
			if err == nil {
				_ = os.Chmod(path, 0o777)
			}
			return nil
		})
	})
	relocated := load([]string{"GOFLAGS=-mod=mod", "GOPROXY=off", "GOMODCACHE=" + relocatedCache})
	if len(baseline) != len(relocated) {
		t.Fatalf("closure sizes differ: %d vs %d", len(baseline), len(relocated))
	}
	for i := range baseline {
		if baseline[i] != relocated[i] {
			t.Errorf("identity %d differs across cache relocation: %q vs %q", i, baseline[i], relocated[i])
		}
	}
}

// TestLoadWorkspaceFailsClosed proves parse errors, type errors, and empty
// matches abort the load; there is no partial universe.
func TestLoadWorkspaceFailsClosed(t *testing.T) {
	broken := t.TempDir()
	writeTree(t, broken, map[string]string{
		"go.mod":  "module example.com/broken\n\ngo 1.26\n",
		"main.go": "package main\n\nfunc main() {",
	})
	if _, err := finalizeLoad(source.Request{Dir: broken}); err == nil {
		t.Error("parse-broken module loaded")
	}
	typeBroken := t.TempDir()
	writeTree(t, typeBroken, map[string]string{
		"go.mod":  "module example.com/typebroken\n\ngo 1.26\n",
		"main.go": "package main\n\nfunc main() { var x int = \"s\"; _ = x }\n",
	})
	if _, err := finalizeLoad(source.Request{Dir: typeBroken}); err == nil {
		t.Error("type-broken module loaded")
	}
	empty := t.TempDir()
	writeTree(t, empty, map[string]string{"go.mod": "module example.com/empty\n\ngo 1.26\n"})
	if _, err := finalizeLoad(source.Request{Dir: empty}); err == nil {
		t.Error("empty module loaded")
	}
}

// TestLoadWorkspaceRejectsModuleCollision proves colliding module identities
// fail closed.
func TestLoadWorkspaceRejectsModuleCollision(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.work":  "go 1.26\n\nuse (\n\t./x\n\t./y\n)\n",
		"x/go.mod": "module clash.example/m\n\ngo 1.26\n",
		"x/x.go":   "package x\n\nfunc X() {}\n",
		"y/go.mod": "module clash.example/m\n\ngo 1.26\n",
		"y/y.go":   "package y\n\nfunc Y() {}\n",
	})
	if _, err := finalizeLoad(source.Request{Dir: dir, Patterns: []string{"clash.example/m/..."}}); err == nil {
		t.Fatal("workspace with colliding module identities loaded")
	}
}

// TestLoadWorkspaceOverlay proves overlays replace on-disk content without
// changing identity or provenance.
func TestLoadWorkspaceOverlay(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod": "module example.com/ol\n\ngo 1.26\n",
		"a.go":   "package a\n\nfunc Broken() {",
	})
	overlay := map[string][]byte{
		filepath.Join(dir, "a.go"): []byte("package a\n\nfunc Fixed() int { return 1 }\n"),
	}
	ws, err := finalizeLoad(source.Request{Dir: dir, Overlay: overlay})
	if err != nil {
		t.Fatalf("LoadWorkspace with overlay: %v", err)
	}
	pkg := ws.Roots()[0]
	if pkg.Types().Scope().Lookup("Fixed") == nil {
		t.Error("overlay content not used")
	}
	if pkg.Provenance() != source.ProvenanceWorkspaceModule {
		t.Errorf("overlay changed provenance: %s", pkg.Provenance())
	}
}
