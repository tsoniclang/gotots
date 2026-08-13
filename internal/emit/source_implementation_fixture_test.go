package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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

func assertSourceImplementationStorageBaseline(
	t *testing.T,
	program *load.Program,
	options Options,
	contractRoots []Root,
	repository string,
	workingDirectory string,
) {
	t.Helper()
	contractSession, err := newProgramSession(program, options)
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := captureSourceImplementationInputs(contractSession, contractRoots)
	if err != nil {
		t.Fatal(err)
	}
	finalSession, err := newProgramSessionWithRegistry(program, options, inputs.registry)
	if err != nil {
		t.Fatal(err)
	}
	finalSession.sourceImplementationContracts = inputs.contracts
	finalSession.sourceImplementationTargets = inputs.targets
	root, err := NewRoot(program.Roots()[0].Types().Scope().Lookup("AZero"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := compileProgramSession(finalSession, []Root{root}, options)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repository, workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	storageCall := false
	for _, file := range emission.Files() {
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			t.Fatal(printErr)
		}
		if strings.Contains(printed, "new Pair__from_fast(") {
			t.Fatalf("certified storage baseline was not selected by the consumer:\n%s", printed)
		}
		storageCall = storageCall || strings.Contains(
			printed,
			"Pair__from_fast.$fromStorage({",
		)
	}
	if !storageCall {
		t.Fatal("certified storage baseline produced no canonical storage call")
	}
}

func assertUncertifiedSourceImplementationRequirementRejected(
	t *testing.T,
	program *load.Program,
	options Options,
	roots []Root,
	pairOwner api.ArtifactOwner,
) {
	t.Helper()
	contractSession, err := newProgramSession(program, options)
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := captureSourceImplementationInputs(contractSession, roots)
	if err != nil {
		t.Fatal(err)
	}
	contracts := make(
		map[api.ArtifactOwner]sourceImplementationContract,
		len(inputs.contracts),
	)
	for owner, contract := range inputs.contracts {
		contracts[owner] = contract
	}
	pairContract, ok := contracts[pairOwner]
	if !ok {
		t.Fatal("source implementation Pair contract is absent")
	}
	retained := make([]api.DeclarationRequirement, 0, len(pairContract.acceptedRequirements))
	removed := false
	for _, requirement := range pairContract.acceptedRequirements {
		_, operation, namedStruct := requirement.NamedStructOperation()
		if namedStruct && operation == api.NamedStructOperationStorage {
			removed = true
			continue
		}
		retained = append(retained, requirement)
	}
	if !removed {
		t.Fatal("source implementation Pair contract has no storage requirement to mutate")
	}
	pairContract.acceptedRequirements = retained
	contracts[pairOwner] = pairContract
	finalSession, err := newProgramSessionWithRegistry(program, options, inputs.registry)
	if err != nil {
		t.Fatal(err)
	}
	finalSession.sourceImplementationContracts = contracts
	finalSession.sourceImplementationTargets = inputs.targets
	if _, err := compileProgramSession(finalSession, roots, options); err == nil ||
		!strings.Contains(err.Error(), "source-implementation requirement was not certified") {
		t.Fatalf("uncertified source-implementation requirement error = %v", err)
	}
}
