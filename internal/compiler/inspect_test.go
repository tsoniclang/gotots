package compiler

import (
	"path/filepath"
	"runtime"
	"testing"

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
		inventory, err := InspectConstructs(source.Request{Dir: dir})
		if err != nil {
			t.Fatalf("%s: %v", project, err)
		}
		d := inventory.Denominators()
		if d.Packages < 3 || d.Files < 3 || d.Occurrences < 200 {
			t.Errorf("%s: implausible denominators %+v", project, d)
		}
		if d.VariantBearing == 0 {
			t.Errorf("%s: no variant-bearing occurrences", project)
		}
	}
}

// TestInspectMultiModuleWorkspace proves a go.work workspace inventories both
// modules with distinct module-qualified identities for identical relative
// paths.
func TestInspectMultiModuleWorkspace(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "testdata", "workspaces", "dual")
	inventory, err := InspectConstructs(source.Request{
		Dir:      dir,
		Patterns: []string{"dual.example/a/...", "dual.example/b/..."},
	})
	if err != nil {
		t.Fatalf("InspectConstructs: %v", err)
	}
	files := map[string]bool{}
	for _, pkg := range inventory.Packages() {
		for _, file := range pkg.Files() {
			files[file.File().String()] = true
		}
	}
	if !files["dual.example/a::pkg/same.go"] || !files["dual.example/b::pkg/same.go"] {
		t.Errorf("identical relative paths not module-qualified: %v", files)
	}
}

// TestInspectSelfModule proves the pipeline runs over a real multi-package
// module: this repository itself.
func TestInspectSelfModule(t *testing.T) {
	inventory, err := InspectConstructs(source.Request{Dir: repoRoot(t)})
	if err != nil {
		t.Fatalf("self-inspection: %v", err)
	}
	d := inventory.Denominators()
	if d.Packages < 7 || d.Occurrences < 5000 {
		t.Errorf("implausible self-inspection denominators %+v", d)
	}
}
