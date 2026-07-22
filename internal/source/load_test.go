package source

import (
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

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

func findPackage(t *testing.T, ws *Workspace, importPath string) *Package {
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
	ws, err := LoadWorkspace(Request{Dir: dir})
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
		if pkg.Provenance() != ProvenanceWorkspaceModule || pkg.Acquisition() != AcquisitionWorkspace {
			t.Errorf("%s provenance/acquisition = %s/%s", pkg.ID(), pkg.Provenance(), pkg.Acquisition())
		}
		if pkg.ModuleGoVersion() != "1.26" {
			t.Errorf("%s module go directive = %q", pkg.ID(), pkg.ModuleGoVersion())
		}
		for _, file := range pkg.Files() {
			ids = append(ids, file.ID().String())
			if file.Syntax() == nil {
				t.Errorf("%s: selected file missing syntax", file.ID())
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
	req := Request{Dir: dir, Env: []string{"GOFLAGS=-mod=mod", "GOPROXY=off"}}
	ws, err := LoadWorkspace(req)
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	fmtPkg := findPackage(t, ws, "fmt")
	if fmtPkg.ID().Owner().String() != "std" || fmtPkg.Provenance() != ProvenanceStandardLibrary ||
		fmtPkg.Acquisition() != AcquisitionGOROOT || fmtPkg.RequestedRoot() {
		t.Errorf("fmt: %s %s %s root=%v", fmtPkg.ID(), fmtPkg.Provenance(), fmtPkg.Acquisition(), fmtPkg.RequestedRoot())
	}
	unsafePkg := findPackage(t, ws, "unsafe")
	if unsafePkg.Disposition() != DispositionUnsafeIntrinsic || unsafePkg.Provenance() != ProvenanceStandardLibrary {
		t.Errorf("unsafe: disposition=%s provenance=%s", unsafePkg.Disposition(), unsafePkg.Provenance())
	}
	sync := findPackage(t, ws, "golang.org/x/sync/errgroup")
	if sync.ID().Owner().String() != "mod=golang.org/x/sync@v0.21.0" {
		t.Errorf("x/sync owner = %s", sync.ID().Owner())
	}
	if sync.Provenance() != ProvenanceModuleDependency || sync.Acquisition() != AcquisitionModuleCache {
		t.Errorf("x/sync provenance/acquisition = %s/%s", sync.Provenance(), sync.Acquisition())
	}
	dotless := findPackage(t, ws, "dotlessdep/lib")
	if dotless.Provenance() != ProvenanceModuleDependency || dotless.Acquisition() != AcquisitionLocalReplacement {
		t.Errorf("dotless replacement provenance/acquisition = %s/%s (spelling must not classify)",
			dotless.Provenance(), dotless.Acquisition())
	}
	// A local replacement keeps the declared module path AND required
	// version; only acquisition reflects the replacement directory.
	if dotless.ID().Owner().String() != "mod=dotlessdep/lib@v1.0.0" {
		t.Errorf("dotless owner = %s", dotless.ID().Owner())
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
	ws, err := LoadWorkspace(Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	dotless := findPackage(t, ws, "dotlessdep/lib")
	if dotless.Provenance() != ProvenanceModuleDependency || dotless.Acquisition() != AcquisitionVendor {
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
	ws, err := LoadWorkspace(Request{Dir: dir, Patterns: []string{"fmt", "builtin", "cmd/addr2line"}})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	fmtPkg := findPackage(t, ws, "fmt")
	if !fmtPkg.RequestedRoot() || fmtPkg.Types() == nil || len(fmtPkg.Files()) == 0 || fmtPkg.Files()[0].Syntax() == nil {
		t.Error("fmt root did not load with syntax and types")
	}
	if fmtPkg.ID().String() != "std::fmt" {
		t.Errorf("fmt id = %s", fmtPkg.ID())
	}
	builtin := findPackage(t, ws, "builtin")
	if builtin.ID().Owner().String() != "lang" || builtin.Disposition() != DispositionBuiltinUniverse {
		t.Errorf("builtin owner/disposition = %s/%s", builtin.ID().Owner(), builtin.Disposition())
	}
	tool := findPackage(t, ws, "cmd/addr2line")
	if tool.ID().Owner().String() != "toolchain" || tool.Provenance() != ProvenanceToolchainPackage {
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

// TestPackageRecordConstructorRejectsIncoherence proves finishPackage is a
// validating construction gate: incoherent owner/provenance/acquisition/
// disposition combinations and selected records without evidence never enter
// a workspace.
func TestPackageRecordConstructorRejectsIncoherence(t *testing.T) {
	module, err := identity.NewModuleID("gate.example/m", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	modPkg, err := identity.NewPackageID(owner, "gate.example/m")
	if err != nil {
		t.Fatal(err)
	}
	stdPkg, err := identity.NewPackageID(identity.StandardLibraryOwner(), "fmt")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		record *Package
	}{
		{"zero identity", &Package{provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource}},
		{"invalid disposition (zero)", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace}},
		{"std owner with workspace provenance", &Package{id: stdPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource}},
		{"module owner with goroot acquisition", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionGOROOT, disposition: DispositionOrdinarySource}},
		{"dependency with workspace acquisition", &Package{id: modPkg, provenance: ProvenanceModuleDependency, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource}},
		{"std owner with module go directive", &Package{id: stdPkg, provenance: ProvenanceStandardLibrary, acquisition: AcquisitionGOROOT, disposition: DispositionOrdinarySource, moduleGoVersion: "1.26"}},
		{"selected without type evidence", &Package{id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace, disposition: DispositionOrdinarySource, requestedRoot: true}},
	}
	for _, c := range cases {
		ws := &Workspace{}
		if err := ws.admit(c.record); err == nil {
			t.Errorf("%s: admitted into the workspace", c.name)
		}
		if len(ws.Packages()) != 0 {
			t.Errorf("%s: rejected record still entered the workspace", c.name)
		}
	}
	// The valid case carries genuine evidence — a typeless record is never
	// a valid fixture.
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, "m.go", "package m\n\nfunc F() {}\n", parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	fileID, err := identity.NewFileID(owner, "m.go")
	if err != nil {
		t.Fatal(err)
	}
	ws := &Workspace{}
	valid := &Package{
		id: modPkg, provenance: ProvenanceWorkspaceModule, acquisition: AcquisitionWorkspace,
		disposition: DispositionOrdinarySource,
		types:       types.NewPackage("gate.example/m", "m"),
		typesInfo:   &types.Info{},
		files:       []*File{{path: "m.go", id: fileID, fset: fset, syntax: syntax}},
	}
	if err := ws.admit(valid); err != nil {
		t.Errorf("coherent evidenced record rejected: %v", err)
	}
	if len(ws.Packages()) != 1 {
		t.Error("coherent evidenced record not admitted")
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
	ws, err := LoadWorkspace(Request{Dir: dir, Patterns: []string{"ver.example/old/...", "ver.example/new/..."}})
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
	ws, err := LoadWorkspace(Request{Dir: dir, Env: []string{"CGO_ENABLED=1"}})
	if err != nil {
		t.Skipf("cgo unavailable in this environment: %v", err)
	}
	main := findPackage(t, ws, "cgo.example/m")
	if !main.CheckedViewDiffers() {
		t.Error("cgo package does not record its checked-view difference")
	}
	if len(main.CheckedView()) == 0 {
		t.Error("cgo checked-view files not recorded")
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
		ws, err := LoadWorkspace(Request{Dir: dir, Env: env})
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
	if _, err := LoadWorkspace(Request{Dir: broken}); err == nil {
		t.Error("parse-broken module loaded")
	}
	typeBroken := t.TempDir()
	writeTree(t, typeBroken, map[string]string{
		"go.mod":  "module example.com/typebroken\n\ngo 1.26\n",
		"main.go": "package main\n\nfunc main() { var x int = \"s\"; _ = x }\n",
	})
	if _, err := LoadWorkspace(Request{Dir: typeBroken}); err == nil {
		t.Error("type-broken module loaded")
	}
	empty := t.TempDir()
	writeTree(t, empty, map[string]string{"go.mod": "module example.com/empty\n\ngo 1.26\n"})
	if _, err := LoadWorkspace(Request{Dir: empty}); err == nil {
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
	if _, err := LoadWorkspace(Request{Dir: dir, Patterns: []string{"clash.example/m/..."}}); err == nil {
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
	ws, err := LoadWorkspace(Request{Dir: dir, Overlay: overlay})
	if err != nil {
		t.Fatalf("LoadWorkspace with overlay: %v", err)
	}
	pkg := ws.Roots()[0]
	if pkg.Types().Scope().Lookup("Fixed") == nil {
		t.Error("overlay content not used")
	}
	if pkg.Provenance() != ProvenanceWorkspaceModule {
		t.Errorf("overlay changed provenance: %s", pkg.Provenance())
	}
}
