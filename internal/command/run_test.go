package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	packageDocument, err := os.ReadFile(filepath.Join(generated, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(packageDocument, []byte(`"type": "module"`)) {
		t.Fatalf("project package = %s", packageDocument)
	}
}

func TestProjectPackageIsRequiredForTopLevelAwait(t *testing.T) {
	root := t.TempDir()
	packageDocument, err := encodeProjectPackage()
	if err != nil {
		t.Fatal(err)
	}
	writeCommandFixture(t, filepath.Join(root, "package.json"), string(packageDocument))
	writeCommandFixture(t, filepath.Join(root, "program.ts"), "await Promise.resolve();\n")
	writeCommandFixture(t, filepath.Join(root, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true
  },
  "files": ["program.ts"]
}
`)
	arguments := []string{"--noEmit", "-p", filepath.Join(root, "tsconfig.json")}
	if err := tsgo.Compile(
		context.Background(),
		repositoryRoot(t),
		root,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "package.json")); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.Compile(
		context.Background(),
		repositoryRoot(t),
		root,
		arguments,
	); err == nil || !strings.Contains(err.Error(), "TS1309") {
		t.Fatalf("CommonJS top-level-await error = %v", err)
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
