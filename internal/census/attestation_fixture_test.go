// Attestation contract fixtures: injected-but-ignored source, tampered
// toolchain pins, and unpinned local module replacements must all be
// rejected before any evidence is published.
package census

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/pinning"
)

func TestCensusRejectsInjectedIgnoredFile(t *testing.T) {
	files := basicFixtureFiles()
	files[".gitignore"] = "injected.go\n"
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	// The file is ignored, so git status stays clean — but it is not part
	// of the pinned commit and must not be admitted.
	injected := filepath.Join(dir, "a", "injected.go")
	if err := os.WriteFile(injected, []byte("package a\n\nfunc Injected() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if clean, err := pinning.CheckClean(dir); err != nil || !clean {
		t.Fatalf("fixture precondition: checkout should look clean (clean=%v err=%v)", clean, err)
	}

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), "injected.go") {
		t.Fatalf("expected failure naming injected.go, got %v", err)
	}
}

func TestCensusRejectsTamperedToolchainPin(t *testing.T) {
	dir, revision := writeFixtureRepo(t, basicFixtureFiles())
	prof := writeFixtureConfig(t, revision, basicFixtureProfile())

	digest := []byte(prof.Pin.Toolchain.GoExecutableSha256)
	if digest[0] == 'a' {
		digest[0] = 'b'
	} else {
		digest[0] = 'a'
	}
	prof.Pin.Toolchain.GoExecutableSha256 = string(digest)

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), "refusing to execute") {
		t.Fatalf("expected digest refusal before execution, got %v", err)
	}
}

func TestCensusRejectsLocalReplacement(t *testing.T) {
	files := map[string]string{
		"go.mod": "module example.com/fix\n\ngo 1.26\n\nrequire example.com/other v0.0.0\n\nreplace example.com/other => ./other\n",
		"a/a.go": `package a

import "example.com/other/lib"

func Use() int { return lib.Value }
`,
		"other/go.mod":     "module example.com/other\n\ngo 1.26\n",
		"other/lib/lib.go": "package lib\n\nconst Value = 1\n",
	}
	dir, revision := writeFixtureRepo(t, files)
	prof := writeFixtureConfig(t, revision, map[string]any{
		"product": "fixture",
		"sourceUniverse": map[string]any{
			"packageRules": []any{map[string]any{
				"id": "selected", "disposition": "selected",
				"selectors": []any{map[string]any{"kind": "subtree", "root": "a"}},
				"overrides": []any{}, "category": "product-source",
				"decision": "FIXTURE", "reason": "census fixture",
			}},
		},
	})

	_, err := Run(prof, dir, "fixture")
	if err == nil || !strings.Contains(err.Error(), "unpinned local replacement") {
		t.Fatalf("expected local-replacement refusal, got %v", err)
	}
}
