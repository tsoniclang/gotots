package policy

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repositoryRoot locates the module root from this test file's position so
// the gate always scans the entire repository regardless of the working
// directory the test runner uses.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate policy test source file")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod found above policy package")
		}
		dir = parent
	}
}

// TestRepositoryFileSizes is the normative repository-wide gate: every
// hand-maintained source file must stay within MaxSourceFileLines.
func TestRepositoryFileSizes(t *testing.T) {
	violations, err := CheckTree(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Error(violation.String())
	}
}

// TestCheckTreeRejectsOversizedFile proves the gate itself detects a
// violation: a synthetic tree containing a file one line over the limit
// must be reported, and a compliant file must not.
func TestCheckTreeRejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	oversized := strings.Repeat("// line\n", MaxSourceFileLines+1)
	for _, name := range []string{"big.go", "big.ts", "big.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(oversized), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	compliant := strings.Repeat("// line\n", MaxSourceFileLines)
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(compliant), 0o644); err != nil {
		t.Fatal(err)
	}
	// Non-source and fixture files are outside the gate.
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(oversized), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "testdata"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "testdata", "fixture.go"), []byte(oversized), 0o644); err != nil {
		t.Fatal(err)
	}

	violations, err := CheckTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 3 {
		t.Fatalf("expected Go, TypeScript, and Markdown violations, got %v", violations)
	}
	for index, want := range []string{"big.go", "big.md", "big.ts"} {
		if violations[index].Path != want || violations[index].Lines != MaxSourceFileLines+1 {
			t.Errorf("violation %d = %+v; want %s at %d lines", index, violations[index], want, MaxSourceFileLines+1)
		}
		if !strings.Contains(violations[index].String(), "GOTOTS_FILE_TOO_LARGE") {
			t.Errorf("diagnostic format wrong: %s", violations[index].String())
		}
	}
}
