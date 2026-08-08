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
	"github.com/tsoniclang/gotots/internal/emit/api"
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
type Reader interface { Read() int }
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "main.go"), `package app
import "example.test/app/fast"
func Value() int {
	current := 41
	return fast.Sum("abcd") + fast.Read(&current)
}
func ReadExisting(current *int) int { return fast.Read(current) }
func IsReader(value any) bool { _, ok := value.(fast.Reader); return ok }
func identity[T any](value T) T { return value }
func GenericValue() int { return identity(42) }
`)
	implementationRoot := filepath.Join(root, "implementation")
	writeSourceImplementationPointerContract(t, implementationRoot)
	implementationSource := `import type { Pointer } from "@tsonic/core/types.js";
import { loadPointer } from "@tsonic/core/lang.js";

export function Read(input: Pointer<number> | undefined): number {
  if (input === undefined) throw new Error("nil pointer");
  return loadPointer(input);
}
interface InterfaceValue {
  $go$implements(contract: readonly object[]): boolean;
}
export interface Reader extends InterfaceValue { Read(): number; }
export const Reader$contract: readonly object[] = Object.freeze([]);
export function Reader$is(value: InterfaceValue | undefined): value is Reader {
  return value !== undefined && value.$go$implements(Reader$contract);
}
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
		Exports: []string{"Read", "Reader", "Reader$contract", "Reader$is", "Sum"},
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
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	options := DefaultOptions()
	options.SourceImplementations = certificate
	contractSession, err := newProgramSession(program, options)
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := captureSourceImplementationInputs(contractSession, roots)
	if err != nil {
		t.Fatal(err)
	}
	fastPackage := program.PackageByPath("example.test/app/fast")
	readerOwner := api.MustSourceArtifactOwner(
		fastPackage.Types().Scope().Lookup("Reader"),
	)
	readerContract, ok := inputs.contracts[readerOwner]
	if !ok {
		t.Fatal("source implementation Reader contract is absent")
	}
	hasMethodToken := false
	for _, requirement := range readerContract.requirements {
		if requirement.Kind() == api.DeclarationRequirementInterfaceMethodToken {
			hasMethodToken = true
			break
		}
	}
	if !hasMethodToken {
		t.Fatal("source implementation Reader discarded its method-token requirement")
	}
	mutatedContracts := make(
		map[api.ArtifactOwner]sourceImplementationContract,
		len(inputs.contracts),
	)
	for owner, contract := range inputs.contracts {
		mutatedContracts[owner] = contract
	}
	readerContract.requirements = nil
	mutatedContracts[readerOwner] = readerContract
	mutationSession, err := newProgramSessionWithRegistry(
		program,
		options,
		inputs.registry,
	)
	if err != nil {
		t.Fatal(err)
	}
	mutationSession.sourceImplementationContracts = mutatedContracts
	mutationSession.sourceImplementationTargets = inputs.targets
	if _, err := compileProgramSession(mutationSession, roots, options); err == nil ||
		!strings.Contains(err.Error(), "dependency provider was not published") {
		t.Fatalf("dropped source-implementation support mutation error = %v", err)
	}
	missingContractSession, err := newProgramSession(program, options)
	if err != nil {
		t.Fatal(err)
	}
	missingContractSession.sourceImplementationContracts =
		make(map[api.ArtifactOwner]sourceImplementationContract)
	if err := missingContractSession.requireProgramRoots(roots); err != nil {
		t.Fatal(err)
	}
	if err := missingContractSession.settle(); err == nil ||
		!strings.Contains(err.Error(), "source-implementation observable contract is absent") {
		t.Fatalf("missing observable-contract mutation error = %v", err)
	}
	emission, err := CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
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
	addressedCaller := false
	identityPointerCaller := false
	interfaceGuardCaller := false
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
			if strings.Contains(printed, "Read__from_fast(addressOf") {
				addressedCaller = true
			}
			if strings.Contains(printed, "Read__from_fast(current)") {
				identityPointerCaller = true
			}
			if strings.Contains(printed, "Reader$is") {
				interfaceGuardCaller = true
			}
			if strings.Contains(printed, "GoPointer") {
				t.Fatalf("caller retained the retired pointer runtime:\n%s", printed)
			}
			if _, generated := generatedPaths[file.OutputPath()]; generated {
				t.Fatalf("generated package file survived: %s", file.OutputPath())
			}
			for generatedPath := range generatedPaths {
				modulePath, moduleErr := output.ModuleSpecifier(
					file.OutputPath(),
					generatedPath,
				)
				if moduleErr != nil {
					t.Fatal(moduleErr)
				}
				if strings.Contains(printed, `"`+modulePath+`"`) {
					t.Fatalf(
						"final artifact %s retained selected-package implementation dependency %s:\n%s",
						file.OutputPath(),
						modulePath,
						printed,
					)
				}
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
	if !addressedCaller {
		t.Fatalf("generated source implementation caller did not pass its canonical address:\n%s", callerArtifacts.String())
	}
	if !identityPointerCaller {
		t.Fatalf("existing pointer caller did not preserve pointer identity:\n%s", callerArtifacts.String())
	}
	if !interfaceGuardCaller {
		t.Fatalf("interface runtime guard did not survive through the package assembly:\n%s", callerArtifacts.String())
	}

	writeSourceImplementationFixture(t, filepath.Join(implementationRoot, "package.ts"), `import type { Pointer } from "@tsonic/core/types.js";
import { loadPointer } from "@tsonic/core/lang.js";
export function Read(value: Pointer<number> | undefined): number {
  if (value === undefined) throw new Error("nil pointer");
  return loadPointer(value);
}
interface InterfaceValue {
  $go$implements(contract: readonly object[]): boolean;
}
export interface Reader extends InterfaceValue { Read(): number; }
export const Reader$contract: readonly object[] = Object.freeze([]);
export function Sum(value: string): number { return value.length; }
`)
	contract.Exports = []string{
		"Read", "Reader", "Reader$contract", "Sum",
	}
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
		!strings.Contains(err.Error(), "private value dependency") {
		t.Fatalf("missing interface-guard mutation error = %v", err)
	}
}

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
			Integers: "number", EvaluationOrder: "direct", Concurrency: "disabled",
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

func writeSourceImplementationFixture(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeSourceImplementationPointerContract(t *testing.T, root string) {
	t.Helper()
	module := filepath.Join(root, "node_modules", "@tsonic", "core")
	writeSourceImplementationFixture(t, filepath.Join(module, "package.json"), `{
  "type": "module",
  "exports": {
    "./lang.js": "./lang.js",
    "./types.js": "./types.js"
  }
}
`)
	writeSourceImplementationFixture(t, filepath.Join(module, "types.d.ts"), `declare const pointerBrand: unique symbol;
export interface Pointer<T> {
  readonly [pointerBrand]: (value: T) => T;
}
`)
	writeSourceImplementationFixture(t, filepath.Join(module, "types.js"), "export {};\n")
	writeSourceImplementationFixture(t, filepath.Join(module, "lang.d.ts"), `import type { Pointer } from "./types.js";
export declare function loadPointer<T>(pointer: Pointer<T>): T;
`)
	writeSourceImplementationFixture(t, filepath.Join(module, "lang.js"), "export {};\n")
}
