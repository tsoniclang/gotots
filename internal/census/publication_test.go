// Publication safety contracts: bidirectional source/output overlap
// rejection, immutable nondestructive bundles, atomic single-winner
// concurrent publication, and manifest verification.
package census

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/inventory"
)

// publishResult builds a minimal publishable Result rooted at sourceDir.
func publishResult(t *testing.T, sourceDir string) *Result {
	t.Helper()
	return &Result{
		Inventory:   &inventory.Inventory{},
		Report:      &Report{SchemaVersion: 3, Production: newScopeReport(), Test: newScopeReport()},
		Environment: &Environment{SourceDir: sourceDir, Toolchain: &goenv.Resolved{}},
		sourceDir:   sourceDir,
	}
}

func makeSourceDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(dir, "workspace", "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	return source
}

func TestPublicationRejectsOverlap(t *testing.T) {
	source := makeSourceDir(t)
	result := publishResult(t, source)

	cases := []struct {
		name string
		out  string
		want string
	}{
		{"equal", source, "is the pinned source tree"},
		{"child", filepath.Join(source, "reports"), "inside the pinned source tree"},
		{"ancestor", filepath.Dir(source), "ancestor of the pinned source tree"},
	}
	for _, c := range cases {
		if err := WriteReports(result, c.out); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: expected %q refusal, got %v", c.name, c.want, err)
		}
	}
}

func TestPublicationRejectsSymlinkOverlap(t *testing.T) {
	source := makeSourceDir(t)
	result := publishResult(t, source)

	// A symlink beside the source pointing into it must not smuggle the
	// bundle inside; a symlink to the source's parent must not hide the
	// ancestor relationship either.
	base := filepath.Dir(source)
	intoSource := filepath.Join(base, "into-source")
	if err := os.Symlink(source, intoSource); err != nil {
		t.Fatal(err)
	}
	if err := WriteReports(result, filepath.Join(intoSource, "reports")); err == nil ||
		!strings.Contains(err.Error(), "pinned source tree") {
		t.Errorf("symlink-into-source overlap not rejected: %v", err)
	}

	toParent := filepath.Join(t.TempDir(), "to-parent")
	if err := os.Symlink(base, toParent); err != nil {
		t.Fatal(err)
	}
	if err := WriteReports(result, toParent); err == nil ||
		!strings.Contains(err.Error(), "ancestor of the pinned source tree") {
		t.Errorf("symlink-to-ancestor overlap not rejected: %v", err)
	}
}

func TestPublicationIsImmutableAndVerifiable(t *testing.T) {
	source := makeSourceDir(t)
	result := publishResult(t, source)
	out := filepath.Join(t.TempDir(), "bundle")

	if err := WriteReports(result, out); err != nil {
		t.Fatalf("publish: %v", err)
	}
	manifest, err := VerifyBundle(out)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(manifest.Files) != 5 {
		t.Errorf("expected 5 manifest files, got %v", manifest.Files)
	}

	// Bundles are immutable: a second publication to the same path is
	// refused and the original is untouched.
	before, err := os.ReadFile(filepath.Join(out, "census.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteReports(result, out); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected immutability refusal, got %v", err)
	}
	after, err := os.ReadFile(filepath.Join(out, "census.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("existing bundle was modified by a refused publication")
	}

	// No staging debris survives.
	entries, err := os.ReadDir(filepath.Dir(out))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".gotots-staging-") {
			t.Errorf("staging directory %s left behind", entry.Name())
		}
	}
}

func TestPublicationConcurrentWritersSingleWinner(t *testing.T) {
	source := makeSourceDir(t)
	out := filepath.Join(t.TempDir(), "bundle")

	const writers = 4
	errs := make([]error, writers)
	var wg sync.WaitGroup
	for i := range writers {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			errs[slot] = WriteReports(publishResult(t, source), out)
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one successful publisher, got %d (errors: %v)", successes, errs)
	}
	if _, err := VerifyBundle(out); err != nil {
		t.Fatalf("winning bundle does not verify: %v", err)
	}
}

func TestVerifyBundleDetectsTampering(t *testing.T) {
	source := makeSourceDir(t)
	out := filepath.Join(t.TempDir(), "bundle")
	if err := WriteReports(publishResult(t, source), out); err != nil {
		t.Fatalf("publish: %v", err)
	}

	censusPath := filepath.Join(out, "census.json")
	data, err := os.ReadFile(censusPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(censusPath, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBundle(out); err == nil || !strings.Contains(err.Error(), "does not match manifest") {
		t.Errorf("tampered file not detected: %v", err)
	}

	if err := os.WriteFile(censusPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "stray.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBundle(out); err == nil || !strings.Contains(err.Error(), "not in the manifest") {
		t.Errorf("stray bundle file not detected: %v", err)
	}
}
