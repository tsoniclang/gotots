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

// TestCgoDependenceBothDirections completes the fixture matrix's per-unit
// C-dependence requirement (verification.md): the existing pipeline fixture
// proves a C-free parent with a C-using nested child; this proves THE INVERSE (a
// C-using parent with a C-free nested child) and C in a function-literal
// signature. C-dependence is per unit, derived from typed evidence, in every
// direction.
func TestCgoDependenceBothDirections(t *testing.T) {
	// usesC's own body calls C.free -> external-boundary; its nested literal is
	// C-free -> full-semantic (the inverse direction). sigC's nested literal has
	// a C type in its SIGNATURE -> external-boundary via the signature alone.
	fixture := "package main\n\n/*\n#include <stdlib.h>\n*/\nimport \"C\"\n\n" +
		"func usesC() {\n\tC.free(nil)\n\tinner := func() int { return 7 }\n\t_ = inner\n}\n\n" +
		"func sigC() {\n\tf := func(p C.int) {}\n\t_ = f\n}\n"

	dir := t.TempDir()
	for rel, content := range map[string]string{
		"go.mod":  "module cgo.example/dir\n\ngo 1.26\n",
		"main.go": fixture,
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
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := analyze.WriteAuditArtifact(artifact, path); err != nil {
		t.Fatal(err)
	}
	req.AuditArtifact = path
	req.AuditArtifactDigest = artifact.ArtifactDigest
	insp, err := InspectConstructs(req)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}

	var pkg *source.Package
	for _, p := range insp.Workspace().Packages() {
		if p.ID().ImportPath() == "cgo.example/dir" {
			pkg = p
		}
	}
	if pkg == nil {
		t.Fatal("cgo package missing")
	}
	byName := map[string]source.SourceUnit{}
	for _, u := range pkg.Units() {
		byName[u.Name()] = u
	}

	// Inverse direction: usesC's body uses C -> external-boundary.
	if got := byName["usesC"].Depth(); got != source.DepthExternalBoundary {
		t.Errorf("usesC (own body uses C) depth = %s, want external-boundary", got)
	}
	// Its nested literal is C-free -> full-semantic (a full child in a non-full
	// cgo parent).
	innerFull := false
	for name, u := range byName {
		if u.Kind() == identity.UnitFuncLitBody && strings.HasPrefix(name, "usesC$lit") {
			if u.Depth() == source.DepthFullSemantic {
				innerFull = true
			}
		}
	}
	if !innerFull {
		t.Error("the C-free nested literal in usesC was not full-semantic — inverse per-unit direction failed")
	}

	// C in a literal signature: sigC's own body is C-free -> full-semantic, but
	// its nested literal's signature names C.int -> external-boundary.
	if got := byName["sigC"].Depth(); got != source.DepthFullSemantic {
		t.Errorf("sigC (own body C-free) depth = %s, want full-semantic", got)
	}
	sigLitCDependent := false
	for name, u := range byName {
		if u.Kind() == identity.UnitFuncLitBody && strings.HasPrefix(name, "sigC$lit") {
			if u.Depth() == source.DepthExternalBoundary {
				sigLitCDependent = true
			}
		}
	}
	if !sigLitCDependent {
		t.Error("the literal with a C type in its signature was not classified C-dependent")
	}
}
