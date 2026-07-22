package stagecheck

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/source"
)

func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return dir
}

var moduleA = map[string]string{
	"go.mod":   "module check.example/a\n\ngo 1.26\n",
	"a.go":     "package a\n\nfunc A(x int) int { return x + 1 }\n",
	"sub/b.go": "package sub\n\nconst B = 2\n",
}

var moduleB = map[string]string{
	"go.mod": "module check.example/b\n\ngo 1.26\n",
	"b.go":   "package b\n\nfunc B() string { return \"b\" }\n",
}

// TestVerifiersPassOnConsistentPipeline proves both stage verifiers accept a
// correctly produced universe and inventory.
func TestVerifiersPassOnConsistentPipeline(t *testing.T) {
	dir := writeModule(t, moduleA)
	req := source.Request{Dir: dir}
	ws, err := source.LoadWorkspace(req)
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if err := VerifySourceUniverse(ws, req); err != nil {
		t.Fatalf("VerifySourceUniverse rejected a consistent universe: %v", err)
	}
	inv, err := analyze.BuildWorkspaceInventory(ws)
	if err != nil {
		t.Fatalf("BuildWorkspaceInventory: %v", err)
	}
	if err := VerifySyntaxInventory(ws, inv); err != nil {
		t.Fatalf("VerifySyntaxInventory rejected a consistent inventory: %v", err)
	}
}

// TestSourceUniverseVerifierIsIndependent is the mutation proof: a workspace
// joined against a different request's toolchain selection fails with exact
// dropped/orphan identities.
func TestSourceUniverseVerifierIsIndependent(t *testing.T) {
	dirA := writeModule(t, moduleA)
	dirB := writeModule(t, moduleB)
	wsA, err := source.LoadWorkspace(source.Request{Dir: dirA})
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	err = VerifySourceUniverse(wsA, source.Request{Dir: dirB})
	if err == nil {
		t.Fatal("verifier accepted a workspace against a different toolchain selection")
	}
	var verification *VerificationError
	if !errors.As(err, &verification) {
		t.Fatalf("error = %T, want *VerificationError", err)
	}
}

// TestSyntaxInventoryVerifierIsIndependent is the mutation proof: an inventory
// joined against a different workspace fails.
func TestSyntaxInventoryVerifierIsIndependent(t *testing.T) {
	dirA := writeModule(t, moduleA)
	dirB := writeModule(t, moduleB)
	wsA, err := source.LoadWorkspace(source.Request{Dir: dirA})
	if err != nil {
		t.Fatalf("LoadWorkspace A: %v", err)
	}
	wsB, err := source.LoadWorkspace(source.Request{Dir: dirB})
	if err != nil {
		t.Fatalf("LoadWorkspace B: %v", err)
	}
	invA, err := analyze.BuildWorkspaceInventory(wsA)
	if err != nil {
		t.Fatalf("BuildWorkspaceInventory: %v", err)
	}
	err = VerifySyntaxInventory(wsB, invA)
	if err == nil {
		t.Fatal("verifier accepted an inventory against a different workspace")
	}
	var verification *VerificationError
	if !errors.As(err, &verification) {
		t.Fatalf("error = %T, want *VerificationError", err)
	}
}

// TestSourceUniverseVerifierHonorsOverlay proves overlayed content verifies
// through the toolchain's own -overlay mechanism.
func TestSourceUniverseVerifierHonorsOverlay(t *testing.T) {
	dir := writeModule(t, map[string]string{
		"go.mod": "module check.example/ol\n\ngo 1.26\n",
		"a.go":   "package a\n\nfunc Broken() {",
	})
	req := source.Request{Dir: dir, Overlay: map[string][]byte{
		filepath.Join(dir, "a.go"): []byte("package a\n\nfunc Fixed() int { return 1 }\n"),
	}}
	ws, err := source.LoadWorkspace(req)
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if err := VerifySourceUniverse(ws, req); err != nil {
		t.Fatalf("VerifySourceUniverse with overlay: %v", err)
	}
}
