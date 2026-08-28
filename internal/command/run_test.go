package command

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestRunPrintsResolvedConfigWithoutBuilding(t *testing.T) {
	root := t.TempDir()
	writeCommandFixture(t, filepath.Join(root, "gotots.json"), `{
  "schemaVersion": 3,
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
  "schemaVersion": 3,
  "distribution": {"root": "`+filepath.ToSlash(repositoryRoot(t))+`"},
  "source": {"root": ".", "package": ".", "mode": "main"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"packages": [], "callables": []},
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
	for _, name := range []string{compileWorkerDirectoryName, protocolScratchDirectoryName} {
		if _, err := os.Stat(filepath.Join(generated, name)); !os.IsNotExist(err) {
			t.Fatalf("successful build retained %s: %v", name, err)
		}
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
	firstReport := output.String()
	firstTree := readGeneratedTree(t, generated)
	output.Reset()
	if err := Run(context.Background(), root, []string{"build"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != firstReport {
		t.Fatalf("ordinary build report changed:\nfirst: %s\nsecond: %s", firstReport, output.String())
	}
	if secondTree := readGeneratedTree(t, generated); !maps.Equal(firstTree, secondTree) {
		t.Fatal("ordinary build artifacts changed across identical source snapshots")
	}
}

func TestRunEmitsCanonicalMarkersWithoutFabricatingCorePackage(t *testing.T) {
	root := t.TempDir()
	writeCommandFixture(t, filepath.Join(root, "go.mod"), "module example.test/markers\n\ngo 1.26.4\n")
	writeCommandFixture(t, filepath.Join(root, "source.go"), `package markers

func Increment(value int32) int32 {
	pointer := &value
	*pointer++
	return value
}
`)
	writeCommandFixture(t, filepath.Join(root, "gotots.json"), `{
  "schemaVersion": 3,
  "distribution": {"root": "`+filepath.ToSlash(repositoryRoot(t))+`"},
  "source": {"root": ".", "package": ".", "mode": "exported"},
  "go": {"goos": "`+runtime.GOOS+`", "goarch": "`+runtime.GOARCH+`", "cgo": false, "tags": []},
  "semantics": {"integers": "number", "evaluationOrder": "direct"},
  "providers": {"standardLibrary": false, "externals": false},
  "implementations": {"packages": [], "callables": []},
  "output": {"directory": "generated"}
}
`)
	generated := filepath.Join(root, "generated")
	writeCommandFixture(t, filepath.Join(generated, "tsconfig.json"), "{}\n")
	writeCommandFixture(t, filepath.Join(generated, "obsolete.ts"), "export {};\n")
	var output bytes.Buffer
	if err := Run(context.Background(), root, []string{"build"}, &output, &output); err != nil {
		t.Fatal(err)
	}
	printed := readGeneratedTypeScript(t, generated)
	if !strings.Contains(printed, `from "@tsonic/core/lang.js"`) ||
		!strings.Contains(printed, `from "@tsonic/core/types.js"`) {
		t.Fatalf("canonical marker imports are absent:\n%s", printed)
	}
	if _, err := os.Stat(filepath.Join(generated, "node_modules", "@tsonic", "core")); !os.IsNotExist(err) {
		t.Fatalf("canonical build fabricated @tsonic/core: %v", err)
	}
	if _, err := os.Stat(filepath.Join(generated, "tsconfig.json")); !os.IsNotExist(err) {
		t.Fatalf("canonical build retained an executable-target tsconfig: %v", err)
	}
	if _, err := os.Stat(filepath.Join(generated, "obsolete.ts")); !os.IsNotExist(err) {
		t.Fatalf("canonical build retained an obsolete owned artifact: %v", err)
	}
	packageDocument, err := os.ReadFile(filepath.Join(generated, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(packageDocument, []byte(`"@gotots/runtime": "0.0.0"`)) ||
		bytes.Contains(packageDocument, []byte(`"@tsonic/core"`)) {
		t.Fatalf("canonical marker dependencies = %s", packageDocument)
	}
	assertManifestMatchesOutput(t, generated)
}

func TestProjectPackageIsRequiredForTopLevelAwait(t *testing.T) {
	root := t.TempDir()
	packageDocument, err := encodeProjectPackage(nil)
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
	if err := tsgo.CompileWithTool(
		context.Background(),
		selectedTSGoTool(t),
		root,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "package.json")); err != nil {
		t.Fatal(err)
	}
	if err := tsgo.CompileWithTool(
		context.Background(),
		selectedTSGoTool(t),
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

func selectedTSGoTool(t *testing.T) tsgo.Tool {
	t.Helper()
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(repositoryRoot(t), ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := tsgo.ResolveTool(selectedGo, repositoryRoot(t), "")
	if err != nil {
		t.Fatal(err)
	}
	return selected
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

func readGeneratedTypeScript(t *testing.T, root string) string {
	t.Helper()
	var source strings.Builder
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".ts" {
			return nil
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source.Write(payload)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return source.String()
}

func readGeneratedTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(relative)] = string(payload)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}

func assertManifestMatchesOutput(t *testing.T, root string) {
	t.Helper()
	payload, err := os.ReadFile(filepath.Join(root, buildManifestName))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SchemaVersion int      `json:"schemaVersion"`
		Files         []string `json:"files"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != buildManifestSchemaVersion {
		t.Fatalf(
			"build manifest schema = %d, want %d",
			manifest.SchemaVersion,
			buildManifestSchemaVersion,
		)
	}
	actual := make([]string, 0, len(manifest.Files))
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		actual = append(actual, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(actual)
	if !slices.Equal(manifest.Files, actual) {
		t.Fatalf("manifest files = %v, physical files = %v", manifest.Files, actual)
	}
}
