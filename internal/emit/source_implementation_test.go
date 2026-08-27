package emit

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func sourceImplementationTestTool(t *testing.T, repository string) tsgo.Tool {
	t.Helper()
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	return selectedTSGo
}

func TestSourceImplementationAtomicallyReplacesGeneratedPackage(t *testing.T) {
	root := t.TempDir()
	writeSourceImplementationFixture(t, filepath.Join(root, "go.mod"), "module example.test/app\n\ngo 1.26.4\n")
	writeSourceImplementationFixture(t, filepath.Join(root, "fast", "fast.go"), `package fast
func Sum(value string) int { return helper(value) * 100 }
func Read(value *int) int { return *value }
type Reader interface { Read() int }
type Pair struct { Left, Right int }
func PairValue() Pair { return Pair{} }
func PairValues() []Pair {
	values := [1]Pair{{}}
	return values[:]
}
`)
	writeSourceImplementationFixture(t, filepath.Join(root, "fast", "helper.go"), `package fast
func helper(value string) int { return len(value) }
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
func AZero() fast.Pair { return fast.Pair{} }
func ZValues() []fast.Pair { return fast.PairValues() }
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
export interface Pair$Storage { Left: number; Right: number; }
export class Pair {
  public constructor(public Left: number, public Right: number) {}
}
export function PairValues(): Pair[] { return [new Pair(0, 0)]; }
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
			Integers: "number", EvaluationOrder: "direct",
		},
		Source:   "package.ts",
		TSConfig: "tsconfig.json",
		Envelope: sourceimplementation.EnvelopeDocument{
			Kind:                 sourceimplementation.EnvelopeInternalAlgorithm,
			RelaxedBehavior:      "internal hash algorithm",
			PreservedObservables: []string{"determinism", "public result shape"},
			Evidence:             []string{"consumer output differential"},
		},
		Exports: []string{
			"Pair", "Pair$Storage", "PairValues", "Read", "Reader",
			"Reader$contract", "Reader$is", "Sum",
		},
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
	sumOwner := api.MustSourceArtifactOwner(
		fastPackage.Types().Scope().Lookup("Sum"),
	)
	helper := fastPackage.Types().Scope().Lookup("helper")
	helperDependency, err := api.NewArtifactDependency(
		api.MustSourceArtifactOwner(helper),
		api.ArtifactFacetImplementation,
	)
	if err != nil {
		t.Fatal(err)
	}
	valueDependency, err := api.NewArtifactDependency(
		api.MustSourceArtifactOwner(
			program.PackageByPath("example.test/app").Types().Scope().Lookup("Value"),
		),
		api.ArtifactFacetImplementation,
	)
	if err != nil {
		t.Fatal(err)
	}
	retained := contractSession.sourceImplementationDependencies(
		[]api.ArtifactDependency{helperDependency, valueDependency},
	)
	if len(retained) != 1 || retained[0] != valueDependency {
		t.Fatal("source-implementation dependency filter lost its ownership boundary")
	}
	for _, dependency := range inputs.contracts[sumOwner].dependencies {
		object, sourceOwned := dependency.Provider().Source()
		if sourceOwned && object.Pkg() == fastPackage.Types() {
			t.Fatalf("captured Sum retained selected-package dependency %s", object.Name())
		}
	}
	readerOwner := api.MustSourceArtifactOwner(
		fastPackage.Types().Scope().Lookup("Reader"),
	)
	readerContract, ok := inputs.contracts[readerOwner]
	if !ok {
		t.Fatal("source implementation Reader contract is absent")
	}
	hasMethodToken := false
	if err := api.WalkRootRequests(
		readerContract.outboundRequests,
		func(request api.RootRequest) error {
			requirement, ok := request.DeclarationRequirement()
			if ok && requirement.Kind() ==
				api.DeclarationRequirementInterfaceMethodToken {
				hasMethodToken = true
			}
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if !hasMethodToken {
		t.Fatal("source implementation Reader discarded its method-token requirement")
	}
	pairOwner := api.MustSourceArtifactOwner(
		fastPackage.Types().Scope().Lookup("Pair"),
	)
	pairContract, ok := inputs.contracts[pairOwner]
	if !ok {
		t.Fatal("source implementation Pair contract is absent")
	}
	hasStorage := false
	for _, requirement := range pairContract.acceptedRequirements {
		_, operation, selected := requirement.NamedStructOperation()
		if selected && operation == api.NamedStructOperationStorage {
			hasStorage = true
			break
		}
	}
	if !hasStorage {
		t.Fatal("source implementation Pair discarded its accepted storage demand")
	}
	assertSourceImplementationStorageBaseline(t, program, options, roots, repository, root)
	assertUncertifiedSourceImplementationRequirementRejected(
		t,
		program,
		options,
		roots,
		pairOwner,
	)
	mutatedContracts := make(
		map[api.ArtifactOwner]sourceImplementationContract,
		len(inputs.contracts),
	)
	for owner, contract := range inputs.contracts {
		mutatedContracts[owner] = contract
	}
	readerContract.outboundRequests = nil
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
	storageLiteralCaller := false
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
			if strings.Contains(printed, "Pair__from_fast.$fromStorage({") {
				storageLiteralCaller = true
			}
			if strings.Contains(printed, "new Pair__from_fast(") {
				t.Fatalf("source implementation caller used the superseded positional representation:\n%s", printed)
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
	if !storageLiteralCaller {
		t.Fatalf("source implementation caller did not select certified canonical storage:\n%s", callerArtifacts.String())
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
export interface Pair$Storage { Left: number; Right: number; }
export class Pair {
  public constructor(public Left: number, public Right: number) {}
}
export function PairValues(): Pair[] { return [new Pair(0, 0)]; }
export function Sum(value: string): number { return value.length; }
`)
	contract.Exports = []string{
		"Pair", "Pair$Storage", "PairValues", "Read", "Reader",
		"Reader$contract", "Sum",
	}
	payload, err = json.MarshalIndent(contract, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeSourceImplementationFixture(t, contractPath, string(payload)+"\n")
	preparedMutation, err := sourceimplementation.PrepareAll(sourceimplementation.Config{
		RepositoryRoot: repository,
		ContractPaths:  []string{contractPath},
		ScratchRoot:    filepath.Join(root, ".mutated-scratch"),
		BuildProfile:   program.BuildProfile(),
		TSGoTool:       sourceImplementationTestTool(t, repository),
		Compilation: sourceimplementation.CompilationDocument{
			Integers: "number", EvaluationOrder: "direct",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mutated, err := preparedMutation.Join(program)
	if err != nil {
		t.Fatal(err)
	}
	options.SourceImplementations = mutated
	if _, err := CompileWithOptions(program, roots, options); err == nil ||
		!strings.Contains(err.Error(), "private value dependency") {
		t.Fatalf("missing interface-guard mutation error = %v", err)
	}
}
