package certify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderPointerContractRejectsSurfaceMutations(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	runtimeDocument, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"contract",
		"runtime.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := runtimecontract.Decode(runtimeDocument)
	if err != nil {
		t.Fatal(err)
	}
	pointerSource, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"src",
		"internal",
		"runtime",
		"pointer.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	original := string(pointerSource)
	mutations := map[string]struct {
		old string
		new string
	}{
		"missing factory": {
			old: "export function providerPointer<T>(value: T): ProviderPointer<T> {\n  return { value };\n}\n",
		},
		"wrong value type": {
			old: "value: T;",
			new: "value: string;",
		},
		"extra member": {
			old: "value: T;",
			new: "value: T;\n  duplicate: T;",
		},
		"wrong factory result": {
			old: "providerPointer<T>(value: T): ProviderPointer<T>",
			new: "providerPointer<T>(value: T): T",
		},
	}
	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			mutated := strings.Replace(original, mutation.old, mutation.new, 1)
			if mutated == original {
				t.Fatal("provider pointer mutation did not alter the fixture")
			}
			providerRoot := t.TempDir()
			pointerPath := filepath.Join(
				providerRoot,
				"src",
				"internal",
				"runtime",
				"pointer.ts",
			)
			if err := os.MkdirAll(filepath.Dir(pointerPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(pointerPath, []byte(mutated), 0o644); err != nil {
				t.Fatal(err)
			}
			tsconfigPath := filepath.Join(providerRoot, "tsconfig.json")
			if err := os.WriteFile(tsconfigPath, []byte(`{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true
  },
  "include": ["src/**/*.ts"]
}
`), 0o644); err != nil {
				t.Fatal(err)
			}
			_, selectedTSGo := resolveTestTools(t, repository)
			client, err := tsgo.StartClientWithTool(selectedTSGo, providerRoot)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := client.Close(); err != nil {
					t.Errorf("close TS-Go client: %v", err)
				}
			})
			project, err := client.OpenProject(tsconfigPath)
			if err != nil {
				t.Fatal(err)
			}
			err = verifyProviderPointerContract(
				resolvedConfig{providerRoot: providerRoot},
				project,
				requirements,
			)
			if err == nil {
				t.Fatal("mutated provider pointer contract was accepted")
			}
		})
	}
}
