package emit

import (
	"context"
	"encoding/json"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
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
func Read(value *int) int { return *value }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "main.go"), `package app
import "example.test/app/fast"
func Value() int {
	current := 41
	return fast.Sum("abcd") + fast.Read(&current)
}
func ReadExisting(current *int) int { return fast.Read(current) }
`)
	implementationRoot := filepath.Join(root, "implementation")
	implementationSource := `export function Read(value: number): number { return value; }
export function Sum(value: string): number { return value.length; }
`
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), implementationSource)
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "tsconfig.json"), `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true
  },
	  "files": ["package.ts"]
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
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
		Source:   "package.ts",
		TSConfig: "tsconfig.json",
		Envelope: sourceimplementation.EnvelopeDocument{
			Kind:                 sourceimplementation.EnvelopeInternalAlgorithm,
			RelaxedBehavior:      "internal hash algorithm",
			PreservedObservables: []string{"determinism", "public result shape"},
			Evidence:             []string{"consumer output differential"},
		},
		Exports: []string{"Read", "Sum"},
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
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	read, ok := program.PackageByPath("example.test/app/fast").Types().Scope().Lookup("Read").(*types.Func)
	if !ok {
		t.Fatal("Go Read function is absent")
	}
	readABI, ok := certificate.ResolveCallableABI(read.Pkg().Path(), read.Name())
	if !ok {
		t.Fatal("Read callable ABI is absent")
	}
	readParameter, ok := readABI.Parameter(0)
	if !ok || readParameter.Projection() != callableabi.ProjectionPointeeValue ||
		readParameter.TargetType() != "number" {
		t.Fatalf("Read callable ABI = %#v", readParameter)
	}
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), `export function Read(value: string): number { return value.length; }
export function Sum(value: string): number { return value.length; }
`)
	if _, err := sourceimplementation.VerifyAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".bad-signature-scratch"),
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
	}); err == nil || !strings.Contains(err.Error(), "join callable ABI") {
		t.Fatalf("unsupported callable projection error = %v", err)
	}
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), implementationSource)
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
	projectedCaller := false
	projectedExistingPointer := false
	var callerArtifacts strings.Builder
	for _, file := range emission.Files() {
		if file.OutputPath() != assemblyPath {
			printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
			if printErr != nil {
				t.Fatal(printErr)
			}
			if strings.Contains(printed, "Read") || strings.Contains(printed, "current") {
				callerArtifacts.WriteString(printed)
			}
			if strings.Contains(printed, "Read__from_fast(current)") {
				projectedCaller = true
				if strings.Contains(printed, "GoPointer.cell") {
					t.Fatalf("projected caller retained a scalar pointer cell:\n%s", printed)
				}
			}
			if strings.Contains(printed, "Read__from_fast(GoPointer.dereference") &&
				strings.Contains(printed, ").value)") {
				projectedExistingPointer = true
			}
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
	if !projectedCaller {
		t.Fatalf("generated source implementation caller did not use pointee-value ABI:\n%s", callerArtifacts.String())
	}
	if !projectedExistingPointer {
		t.Fatalf("existing pointer caller did not read its current pointee once:\n%s", callerArtifacts.String())
	}

	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), `export function Read(value: number): string { return String(value); }
export function Sum(value: string): number { return value.length; }
`)
	resultMismatch, err := sourceimplementation.VerifyAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".result-mismatch-scratch"),
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	options.SourceImplementations = resultMismatch
	if _, err := CompileWithOptions(program, roots, options); err == nil ||
		!strings.Contains(err.Error(), "fast.Read") ||
		!strings.Contains(err.Error(), "surface differs") {
		t.Fatalf("result-contract mutation error = %v", err)
	}
	writeSourceImplementationFixture(
		t,
		filepath.Join(implementationRoot, "package.ts"),
		implementationSource,
	)
	options.SourceImplementations = certificate

	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), `export function Extra(): number { return 0; }
export function Read(value: number): number { return value; }
export function Sum(value: string): number { return value.length; }
`)
	contract.Exports = []string{"Extra", "Read", "Sum"}
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
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	options.SourceImplementations = mutated
	if _, err := CompileWithOptions(program, roots, options); err == nil ||
		!strings.Contains(err.Error(), "generated exports") {
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
