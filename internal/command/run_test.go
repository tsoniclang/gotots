package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunPrintsResolvedConfigWithoutBuilding(t *testing.T) {
	root := t.TempDir()
	writeCommandFixture(t, filepath.Join(root, "gotots.json"), `{
  "schemaVersion": 1,
  "distribution": {"root": "`+filepath.ToSlash(repositoryRoot(t))+`"},
  "source": {"root": ".", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "output": {"directory": "generated"}
}
`)
	var output bytes.Buffer
	if err := Run(context.Background(), root, []string{
		"build", "--print-resolved-config",
	}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"integers": "number"`) ||
		!strings.Contains(output.String(), `"directory": "`+filepath.ToSlash(filepath.Join(root, "generated"))+`"`) {
		t.Fatalf("resolved config = %s", output.String())
	}
}

func TestRunBuildsSimpleProgramThroughPinnedTSGo(t *testing.T) {
	root := t.TempDir()
	writeCommandFixture(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo 1.26.4\n")
	writeCommandFixture(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}\n")
	writeCommandFixture(t, filepath.Join(root, "gotots.json"), `{
  "schemaVersion": 1,
  "distribution": {"root": "`+filepath.ToSlash(repositoryRoot(t))+`"},
  "source": {"root": ".", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct", "concurrency": "disabled"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"bundles": []},
  "output": {"directory": "generated"}
}
`)
	var output bytes.Buffer
	if err := Run(context.Background(), root, []string{"build"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(root, "generated")
	entries, err := os.ReadDir(generated)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("build emitted no target artifacts")
	}
	manifest, err := os.ReadFile(filepath.Join(generated, "gotots-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(manifest, []byte(`"semanticDigest"`)) {
		t.Fatalf("build manifest = %s", manifest)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func writeCommandFixture(t *testing.T, path string, source string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}
