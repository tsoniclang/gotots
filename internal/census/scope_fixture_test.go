// Scope-boundary contract: outside-universe roots are filtered before
// census, and a selected dependency into one is a blocking scope error —
// never a contradiction record, an external obligation, or a silent
// reclassification.
package census

import (
	"strings"
	"testing"
)

// scopeViolationFixtureFiles is the basic fixture plus one selected
// production import into the outside-universe root.
func scopeViolationFixtureFiles() map[string]string {
	files := basicFixtureFiles()
	files["c/c.go"] = `package c

import "example.com/fix/ex"

func UseOutside() { ex.Touch() }
`
	files["ex/ex.go"] = `package ex

func Touch() {}
`
	return files
}

func TestScopeDependencyOutsideFailsClosed(t *testing.T) {
	dir, revision := writeFixtureRepo(t, scopeViolationFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	_, err := Run(prof, dir, "fixture")
	if err == nil {
		t.Fatal("selected dependency into an outside-universe root must fail closed")
	}
	if !strings.Contains(err.Error(), "GOTOTS_SCOPE_DEPENDENCY_OUTSIDE") {
		t.Fatalf("expected GOTOTS_SCOPE_DEPENDENCY_OUTSIDE, got: %v", err)
	}
	if !strings.Contains(err.Error(), "example.com/fix/c") || !strings.Contains(err.Error(), "example.com/fix/ex") {
		t.Fatalf("scope diagnostic must name the selected package and the outside root: %v", err)
	}
	if !strings.Contains(err.Error(), "editor-service") {
		t.Fatalf("scope diagnostic must name the outside-universe category: %v", err)
	}
}

func TestScopeDependencyOutsideFromTestScope(t *testing.T) {
	files := basicFixtureFiles()
	files["ex/ex.go"] = `package ex

func Touch() {}
`
	files["c/c_test.go"] = `package c

import "example.com/fix/ex"

func useOutsideInTest() { ex.Touch() }
`
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	_, err := Run(prof, dir, "fixture")
	if err == nil {
		t.Fatal("selected test dependency into an outside-universe root must fail closed")
	}
	if !strings.Contains(err.Error(), "GOTOTS_SCOPE_DEPENDENCY_OUTSIDE") {
		t.Fatalf("expected GOTOTS_SCOPE_DEPENDENCY_OUTSIDE, got: %v", err)
	}
	if !strings.Contains(err.Error(), "test") {
		t.Fatalf("scope diagnostic must name the importing scope: %v", err)
	}
}
