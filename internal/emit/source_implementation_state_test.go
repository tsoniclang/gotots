package emit

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestSourceImplementationOwnsPrivateStateAndInitialization(t *testing.T) {
	root := t.TempDir()
	writeSourceImplementationFixture(
		t,
		filepath.Join(root, "go.mod"),
		"module example.test/app\n\ngo 1.26.4\n",
	)
	writeSourceImplementationFixture(t, filepath.Join(root, "fast", "fast.go"), `package fast

import "unsafe"

var private = unsafe.Pointer(new(int))
var Public = 7

type State struct {
	private unsafe.Pointer
	Value int
}

func NewState(value int) State {
	return State{private: private, Value: value}
}

func Sum(value string) int { return len(value) * 100 }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "main.go"), `package app

import "example.test/app/fast"

func Value() int { return fast.Sum("abcd") + fast.NewState(1).Value + fast.Public }
`)
	implementationRoot := filepath.Join(root, "implementation")
	writeSourceImplementationFixture(
		t,
		filepath.Join(implementationRoot, "package.ts"),
		`export class State {
  public constructor(public Value: number) {}
}
export function $initialize(): void {}
export const $state = { Public: 7 };
export function NewState(value: number): State { return new State(value); }
export function Sum(value: string): number { return value.length; }
`,
	)
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
			Integers: "number", EvaluationOrder: "direct",
		},
		Source:   "package.ts",
		TSConfig: "tsconfig.json",
		Envelope: sourceimplementation.EnvelopeDocument{
			Kind: sourceimplementation.EnvelopeExact,
		},
		Exports: []string{"$initialize", "$state", "NewState", "State", "Sum"},
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
	prepared, err := sourceimplementation.PrepareAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".scratch"),
		BuildProfile:   program.BuildProfile(),
		TSGoTool:       sourceImplementationTestTool(t, repository),
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := prepared.Join(program)
	if err != nil {
		t.Fatal(err)
	}
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := DefaultOptions()
	options.IntegerRepresentation = IntegerRepresentationNumber
	options.EvaluationOrder = EvaluationOrderDirect
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
	client, err := tsgo.StartClient(repository, root)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, file := range emission.Files() {
		if file.OutputPath() != assemblyPath {
			continue
		}
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			t.Fatal(printErr)
		}
		if file.Kind() != TargetFileSourceImplementation ||
			strings.Contains(printed, "private") ||
			!strings.Contains(printed, "return value.length;") {
			t.Fatalf("source implementation package =\n%s", printed)
		}
		return
	}
	t.Fatal("source implementation package assembly is absent")
}
