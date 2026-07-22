package source

import (
	"os"
	"path/filepath"
	"testing"
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

// TestLoadWorkspaceSingleModule proves a module loads into typed packages with
// module-relative machine-independent file identities and type information.
func TestLoadWorkspaceSingleModule(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod":       "module example.com/single\n\ngo 1.26\n",
		"main.go":      "package main\n\nfunc main() { _ = add(1, 2) }\n\nfunc add(a, b int) int { return a + b }\n",
		"pkg/util.go":  "package pkg\n\n// Twice doubles.\nfunc Twice(x int) int { return 2 * x }\n",
		"pkg/util2.go": "package pkg\n\nconst K = 3\n",
	})
	ws, err := LoadWorkspace(Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if len(ws.Packages()) != 2 {
		t.Fatalf("loaded %d packages, want 2", len(ws.Packages()))
	}
	var ids []string
	for _, pkg := range ws.Packages() {
		if pkg.Types() == nil || pkg.TypesInfo() == nil {
			t.Errorf("%s: missing type information", pkg.ID())
		}
		for _, file := range pkg.Files() {
			ids = append(ids, file.ID().String())
			if file.Syntax() == nil {
				t.Errorf("%s: missing syntax", file.ID())
			}
		}
	}
	want := []string{
		"example.com/single::main.go",
		"example.com/single::pkg/util.go",
		"example.com/single::pkg/util2.go",
	}
	if len(ids) != len(want) {
		t.Fatalf("file identities %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("file identity %d = %q, want %q", i, ids[i], want[i])
		}
	}
}

// TestLoadWorkspaceMultiModule proves a go.work workspace distinguishes two
// modules containing identical relative paths.
func TestLoadWorkspaceMultiModule(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.work":       "go 1.26\n\nuse (\n\t./a\n\t./b\n)\n",
		"a/go.mod":      "module example.com/a\n\ngo 1.26\n",
		"a/pkg/same.go": "package pkg\n\nfunc A() int { return 1 }\n",
		"b/go.mod":      "module example.com/b\n\ngo 1.26\n",
		"b/pkg/same.go": "package pkg\n\nfunc B() int { return 2 }\n",
	})
	// A go.work root holds no module of its own; workspace-wide selection
	// names the member modules (the go tool's own pattern semantics).
	ws, err := LoadWorkspace(Request{Dir: dir, Patterns: []string{"example.com/a/...", "example.com/b/..."}})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	seen := map[string]bool{}
	for _, pkg := range ws.Packages() {
		for _, file := range pkg.Files() {
			seen[file.ID().String()] = true
			if file.ID().Rel() != "pkg/same.go" {
				t.Errorf("unexpected rel %q", file.ID().Rel())
			}
		}
	}
	if !seen["example.com/a::pkg/same.go"] || !seen["example.com/b::pkg/same.go"] {
		t.Errorf("identical relative paths did not receive distinct module-qualified identities: %v", seen)
	}
	if len(seen) != 2 {
		t.Errorf("expected exactly 2 file identities, got %v", seen)
	}
}

// TestLoadWorkspaceRelocatedIdentity proves identity survives checkout
// relocation: the same module content in two directories yields identical
// identities.
func TestLoadWorkspaceRelocatedIdentity(t *testing.T) {
	content := map[string]string{
		"go.mod":     "module example.com/reloc\n\ngo 1.26\n",
		"pkg/x.go":   "package pkg\n\nfunc X() {}\n",
		"pkg/sub.go": "package pkg\n\nvar V = 1\n",
	}
	load := func() []string {
		dir := t.TempDir()
		writeTree(t, dir, content)
		ws, err := LoadWorkspace(Request{Dir: dir})
		if err != nil {
			t.Fatalf("LoadWorkspace: %v", err)
		}
		var ids []string
		for _, pkg := range ws.Packages() {
			ids = append(ids, pkg.ID().String())
			for _, f := range pkg.Files() {
				ids = append(ids, f.ID().String())
			}
		}
		return ids
	}
	a, b := load(), load()
	if len(a) != len(b) {
		t.Fatalf("identity sets differ in size: %v vs %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("identity %d differs across checkouts: %q vs %q", i, a[i], b[i])
		}
	}
}

// TestLoadWorkspaceFailsClosed proves parse errors, type errors, and empty
// matches abort the load; there is no partial universe.
func TestLoadWorkspaceFailsClosed(t *testing.T) {
	parseBroken := t.TempDir()
	writeTree(t, parseBroken, map[string]string{
		"go.mod":  "module example.com/broken\n\ngo 1.26\n",
		"main.go": "package main\n\nfunc main() {",
	})
	if _, err := LoadWorkspace(Request{Dir: parseBroken}); err == nil {
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
	writeTree(t, empty, map[string]string{
		"go.mod": "module example.com/empty\n\ngo 1.26\n",
	})
	if _, err := LoadWorkspace(Request{Dir: empty}); err == nil {
		t.Error("empty module loaded")
	}
}

// TestLoadWorkspaceRejectsModuleCollision proves two modules claiming the
// same module path cannot form one universe: the load fails closed (either
// through the toolchain's own rejection or the loader's identity-collision
// check), never producing ambiguous identities.
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

// TestLoadWorkspaceOverlay proves overlays replace on-disk content.
func TestLoadWorkspaceOverlay(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"go.mod": "module example.com/ol\n\ngo 1.26\n",
		"a.go":   "package a\n\nfunc Broken() {", // broken on disk
	})
	overlay := map[string][]byte{
		filepath.Join(dir, "a.go"): []byte("package a\n\nfunc Fixed() int { return 1 }\n"),
	}
	ws, err := LoadWorkspace(Request{Dir: dir, Overlay: overlay})
	if err != nil {
		t.Fatalf("LoadWorkspace with overlay: %v", err)
	}
	pkg := ws.Packages()[0]
	if pkg.Types().Scope().Lookup("Fixed") == nil {
		t.Error("overlay content not used")
	}
}
