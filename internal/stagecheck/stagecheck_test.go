package stagecheck

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// mustContract resolves the default contract artifact (the test's request
// selection).
func mustContract() scope.ProviderContract {
	contract, err := scope.ResolveContract(scope.DefaultContractID, "", "")
	if err != nil {
		panic(err)
	}
	return contract
}

// loadFinalized runs the full source pipeline under the default contract.
func loadFinalized(req source.Request) (*source.Workspace, error) {
	contract := mustContract()
	policy, err := contract.AuditAcquisitionPolicy()
	if err != nil {
		return nil, err
	}
	universe, err := source.LoadUniverse(req, policy, source.UnitManifest{})
	if err != nil {
		return nil, err
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		return nil, err
	}
	return source.Finalize(universe, selection.Depths(), selection.ImplicitDepths())
}

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
	"go.mod":   "module check.example/a\n\ngo 1.24\n",
	"a.go":     "package a\n\nimport (\n\t\"embed\"\n\t\"fmt\"\n\t\"unsafe\"\n)\n\n//go:embed note.txt\nvar Note embed.FS\n\nfunc A(x int) int { return x + int(unsafe.Sizeof(0)) }\n\nfunc Show(x int) string { return fmt.Sprint(x) }\n",
	"newer.go": "//go:build go1.26\n\npackage a\n\nfunc Newer() int { return 2 }\n",
	"asm.s":    "// reference assembly input (no symbols)\n",
	"note.txt": "embedded note\n",
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
	ws, err := loadFinalized(req)
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
	wsA, err := loadFinalized(source.Request{Dir: dirA})
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
	wsA, err := loadFinalized(source.Request{Dir: dirA})
	if err != nil {
		t.Fatalf("LoadWorkspace A: %v", err)
	}
	wsB, err := loadFinalized(source.Request{Dir: dirB})
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

// TestVerifierUsesSelectedToolchain proves the loader and verifier execute
// the exact selected binary, never an ambient `go` re-resolution: a logging
// wrapper is selected, and both the loader's pattern-set runs and the
// verifier's -deps join appear in its invocation log.
func TestVerifierUsesSelectedToolchain(t *testing.T) {
	real, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go binary")
	}
	dir := writeModule(t, moduleA)
	logFile := filepath.Join(t.TempDir(), "invocations.log")
	// The selected binary is deliberately NOT named "go": the loader's shim
	// must still route the go/packages driver through it.
	wrapper := filepath.Join(t.TempDir(), "custom-toolchain.sh")
	script := "#!/bin/sh\necho \"$@\" >> " + logFile + "\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	req := source.Request{Dir: dir, GoBinary: wrapper}
	ws, err := loadFinalized(req)
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if ws.Toolchain().Binary() != wrapper {
		t.Fatalf("toolchain binary = %s, want wrapper", ws.Toolchain().Binary())
	}
	if err := VerifySourceUniverse(ws, req); err != nil {
		t.Fatalf("VerifySourceUniverse: %v", err)
	}
	logged, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	for _, needle := range []string{"list std", "list cmd", "list -deps -json"} {
		if !strings.Contains(string(logged), needle) {
			t.Errorf("selected toolchain never ran %q; log:\n%s", needle, logged)
		}
	}
	// The go/packages driver itself must have gone through the selected
	// binary (multiple list invocations beyond the three direct ones).
	if strings.Count(string(logged), "list") < 4 {
		t.Errorf("go/packages driver bypassed the selected toolchain; log:\n%s", logged)
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
	ws, err := loadFinalized(req)
	if err != nil {
		t.Fatalf("LoadWorkspace: %v", err)
	}
	if err := VerifySourceUniverse(ws, req); err != nil {
		t.Fatalf("VerifySourceUniverse with overlay: %v", err)
	}
}
