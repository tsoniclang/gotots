// Build-input injection fixtures: every input class any inventoried
// package can name — build-tag-excluded Go files, non-Go build inputs,
// embed payloads, and files in unselected packages — must be attested
// against the pinned commit tree even though none of them is type-checked.
// An untracked injection in any class fails the census while git status
// stays clean.
package census

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/pinning"
)

// runInjection commits the fixture with extra tracked files, then injects
// an ignored (untracked) file and expects the census to fail naming it.
func runInjection(t *testing.T, extraTracked map[string]string, injectPath, injectContent string) {
	t.Helper()
	files := basicFixtureFiles()
	files[".gitignore"] = filepath.Base(injectPath) + "\n"
	for path, content := range extraTracked {
		files[path] = content
	}
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	full := filepath.Join(dir, filepath.FromSlash(injectPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(injectContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean, err := pinning.CheckClean(dir); err != nil || !clean {
		t.Fatalf("fixture precondition: checkout should look clean (clean=%v err=%v)", clean, err)
	}

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), filepath.Base(injectPath)) {
		t.Fatalf("expected failure naming %s, got %v", injectPath, err)
	}
}

func TestInjectionBuildTagExcludedGoFile(t *testing.T) {
	// The injected file is excluded by a build constraint, so it is never
	// type-checked — it lands in IgnoredGoFiles and must still be attested.
	runInjection(t, nil, "a/tagged_injected.go", "//go:build neverbuildme\n\npackage a\n")
}

func TestInjectionNonGoBuildInput(t *testing.T) {
	// Assembly is a non-Go build input (SFiles); with cgo disabled and no
	// matching build it is still named by go list and must be attested.
	runInjection(t, nil, "a/asm_injected.s", "// injected assembly\n")
}

func TestInjectionEmbeddedFile(t *testing.T) {
	// The embed pattern matches a whole directory: an untracked file
	// appearing there becomes an embed payload without any Go change.
	extra := map[string]string{
		"a/embedder.go": `package a

import "embed"

//go:embed payload
var payloadFS embed.FS
`,
		"a/payload/data.txt": "tracked payload\n",
	}
	runInjection(t, extra, "a/payload/extra_injected.txt", "injected payload\n")
}

func TestInjectionOutsideUniverseRootIsInert(t *testing.T) {
	// Outside-universe roots are filtered before census: nothing below
	// them is a census input, so an injected file there cannot affect any
	// selected record — the run succeeds and the file has no disposition.
	files := basicFixtureFiles()
	files[".gitignore"] = "ex_injected.go\n"
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	full := filepath.Join(dir, "ex", "ex_injected.go")
	if err := os.WriteFile(full, []byte("package ex\n\nfunc Injected() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Run(prof, dir, "fixture")
	if err != nil {
		t.Fatalf("injection below an outside-universe root must be inert: %v", err)
	}
	// Only the tracked outside file is counted by the filter evidence.
	if result.Inventory.Universe.OutsideUniverseFiles != 1 {
		t.Errorf("untracked injected file must not enter filter evidence: %+v", result.Inventory.Universe.OutsideUniverse)
	}
}
