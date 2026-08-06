package emit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSourceImplementationAtomicallyReplacesGeneratedPackage(t *testing.T) {
	root := t.TempDir()
	writeSourceImplementationFixture(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo 1.26.4\n")
	writeSourceImplementationFixture(t, filepath.Join(root, "fast", "fast.go"), `package fast
func Sum(value string) int { return len(value) * 100 }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "main.go"), `package app
import "example.test/app/fast"
func Value() int { return fast.Sum("abcd") }
`)
	implementationRoot := filepath.Join(root, "implementation")
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), `export function Sum(value: string): number { return value.length; }
`)
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true
  },
  "include": ["package.ts"]
}
`)
	contract := sourceimplementation.Document{
		SchemaVersion: sourceimplementation.SchemaVersion,
		Package: sourceimplementation.PackageDocument{
			ImportPath: "example.test/app/fast",
			ModulePath: "example.test/app",
		},
		Build: sourceimplementation.BuildDocument{
			GoVersion: runtime.Version(),
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
		},
		Source:   "package.ts",
		TSConfig: "tsconfig.json",
		Envelope: sourceimplementation.EnvelopeDocument{
			Kind:                 sourceimplementation.EnvelopeInternalAlgorithm,
			RelaxedBehavior:      "internal hash algorithm",
			PreservedObservables: []string{"determinism", "public result shape"},
			Evidence:             []string{"consumer output differential"},
		},
		Exports: []string{"Sum"},
	}
	payload, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(implementationRoot, "contract.json")
	writeSourceImplementationFixture(t, contractPath, string(payload)+"\n")

	program, err := load.Load(context.Background(), load.Request{
		Directory: root,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := sourceimplementation.VerifyAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".scratch"),
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := DefaultOptions()
	options.SourceImplementations = certificate
	emission, err := CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	fastPackage := program.PackageByPath("example.test/app/fast")
	assemblyPath, err := output.PackageAssemblyPath(fastPackage)
	if err != nil {
		t.Fatal(err)
	}
	statePath, err := output.PackageStatePath(fastPackage)
	if err != nil {
		t.Fatal(err)
	}
	generatedPaths := map[string]struct{}{statePath: {}}
	for _, sourceFile := range fastPackage.Files() {
		outputPath, pathErr := output.SourcePath(fastPackage, sourceFile)
		if pathErr != nil {
			t.Fatal(pathErr)
		}
		generatedPaths[outputPath] = struct{}{}
	}
	client, err := tsgo.StartClient(repository, root)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	selected := 0
	for _, file := range emission.Files() {
		if file.OutputPath() != assemblyPath {
			if _, generated := generatedPaths[file.OutputPath()]; generated {
				t.Fatalf("generated package file survived: %s", file.OutputPath())
			}
			continue
		}
		selected++
		if file.Kind() != TargetFileSourceImplementation {
			t.Fatalf("replacement kind is %v", file.Kind())
		}
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			t.Fatal(printErr)
		}
		if !strings.Contains(printed, "return value.length;") ||
			strings.Contains(printed, "value.length * 100") {
			t.Fatalf("wrong package owner printed:\n%s", printed)
		}
	}
	if selected != 1 {
		t.Fatalf("manual package materialized %d times", selected)
	}

	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), `export function Extra(): number { return 0; }
export function Sum(value: string): number { return value.length; }
`)
	contract.Exports = []string{"Extra", "Sum"}
	payload, err = json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeSourceImplementationFixture(t, contractPath, string(payload)+"\n")
	mutated, err := sourceimplementation.VerifyAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".mutated-scratch"),
	})
	if err != nil {
		t.Fatal(err)
	}
	options.SourceImplementations = mutated
	if _, err := CompileWithOptions(program, roots, options); err == nil ||
		!strings.Contains(err.Error(), "exports [Extra Sum] differ from generated surface [Sum]") {
		t.Fatalf("generated-surface mutation error = %v", err)
	}
}

func writeSourceImplementationFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
