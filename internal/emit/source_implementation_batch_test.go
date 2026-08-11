package emit

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestSourceImplementationBundlesInstallAsOneTransaction(t *testing.T) {
	root := t.TempDir()
	writeSourceImplementationFixture(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo 1.26.4\n")
	writeSourceImplementationFixture(t, filepath.Join(root, "alpha", "alpha.go"), `package alpha
func Add(left, right int) int { return left + right }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "beta", "beta.go"), `package beta
func Twice(value int) int { return value * 2 }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "main.go"), `package app
import (
	"example.test/app/alpha"
	"example.test/app/beta"
)
func Value() int { return beta.Twice(alpha.Add(20, 1)) }
`)

	alphaContract := writeBatchSourceImplementation(
		t,
		root,
		"alpha-implementation",
		"example.test/app/alpha",
		"export function Add(left: number, right: number): number { return left + right; }\n",
		[]string{"Add"},
	)
	betaContract := writeBatchSourceImplementation(
		t,
		root,
		"beta-implementation",
		"example.test/app/beta",
		"export function Twice(value: number): number { return value + value; }\n",
		[]string{"Twice"},
	)

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
	compilation := sourceimplementation.CompilationDocument{
		Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
	}
	certificate, err := sourceimplementation.VerifyAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		Program:        program,
		ContractPaths:  []string{alphaContract, betaContract},
		ScratchRoot:    filepath.Join(root, ".scratch"),
		TSGoTool:       sourceImplementationTestTool(t, repository),
		Compilation:    compilation,
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

	assemblyPaths := make(map[string]struct{}, 2)
	retiredPaths := make(map[string]struct{})
	for _, packagePath := range []string{"example.test/app/alpha", "example.test/app/beta"} {
		selectedPackage := program.PackageByPath(packagePath)
		assemblyPath, pathErr := output.PackageAssemblyPath(selectedPackage)
		if pathErr != nil {
			t.Fatal(pathErr)
		}
		assemblyPaths[assemblyPath] = struct{}{}
		statePath, pathErr := output.PackageStatePath(selectedPackage)
		if pathErr != nil {
			t.Fatal(pathErr)
		}
		retiredPaths[statePath] = struct{}{}
		for _, sourceFile := range selectedPackage.Files() {
			sourcePath, sourceErr := output.SourcePath(selectedPackage, sourceFile)
			if sourceErr != nil {
				t.Fatal(sourceErr)
			}
			retiredPaths[sourcePath] = struct{}{}
		}
	}
	installed := make(map[string]struct{}, 2)
	for _, file := range emission.Files() {
		if _, retired := retiredPaths[file.OutputPath()]; retired {
			t.Fatalf("selected generated package artifact survived: %s", file.OutputPath())
		}
		if _, selected := assemblyPaths[file.OutputPath()]; !selected {
			continue
		}
		if file.Kind() != TargetFileSourceImplementation {
			t.Fatalf("selected assembly %s has kind %v", file.OutputPath(), file.Kind())
		}
		installed[file.OutputPath()] = struct{}{}
	}
	if len(installed) != len(assemblyPaths) {
		t.Fatalf("installed source implementation assemblies = %d, want %d", len(installed), len(assemblyPaths))
	}
}

func writeBatchSourceImplementation(
	t *testing.T,
	root string,
	directory string,
	packagePath string,
	source string,
	exports []string,
) string {
	t.Helper()
	implementationRoot := filepath.Join(root, directory)
	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), source)
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
	document := sourceimplementation.Document{
		SchemaVersion: sourceimplementation.SchemaVersion,
		Package: sourceimplementation.PackageDocument{
			ImportPath: packagePath,
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
		Envelope: sourceimplementation.EnvelopeDocument{Kind: sourceimplementation.EnvelopeExact},
		Exports:  exports,
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	contractPath := filepath.Join(implementationRoot, "contract.json")
	writeSourceImplementationFixture(t, contractPath, string(payload)+"\n")
	return contractPath
}
