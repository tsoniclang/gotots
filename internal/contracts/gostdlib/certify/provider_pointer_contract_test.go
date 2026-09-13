package certify

import (
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
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

func TestSourceRawPointerContractRequiresExactCanonicalIdentity(test *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		test.Fatal(err)
	}
	root := test.TempDir()
	write := func(name, text string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			test.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			test.Fatal(err)
		}
	}
	if err := corefixture.InstallResolutionOnly(root); err != nil {
		test.Fatal(err)
	}
	write("package.json", `{"type":"module"}`)
	write("tsconfig.json", `{"compilerOptions":{"target":"ES2022","module":"NodeNext","moduleResolution":"NodeNext","strict":true},"include":["*.ts","test/**/*.ts"]}`)
	write("surface.ts", `
import type { RawPointer } from "@tsonic/core/types.js";
interface Wrong extends RawPointer {}
export declare function Good(value: RawPointer | undefined): RawPointer | undefined;
export declare function WrongInput(value: Wrong | undefined): RawPointer | undefined;
export declare function WrongOutput(value: RawPointer | undefined): Wrong | undefined;
export declare function Pair(value: RawPointer | undefined): [RawPointer | undefined, RawPointer | undefined];
export declare function WrongPair(value: RawPointer | undefined): [RawPointer | undefined, Wrong | undefined];
export declare function NotTuple(value: RawPointer | undefined): (RawPointer | undefined)[];
export declare function StaticGood(receiver: number, value: RawPointer | undefined): RawPointer | undefined;
export declare function StaticWrong(receiver: RawPointer | undefined, value: Wrong | undefined): RawPointer | undefined;
`)
	_, tool := resolveTestTools(test, repository)
	client, err := tsgo.StartClientWithTool(tool, root)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := client.Close(); err != nil {
			test.Error(err)
		}
	})
	project, err := client.OpenProject(filepath.Join(root, "tsconfig.json"))
	if err != nil {
		test.Fatal(err)
	}
	exports, err := project.Exports(filepath.Join(root, "surface.ts"))
	if err != nil {
		test.Fatal(err)
	}
	if len(exports) != 8 {
		test.Fatalf("raw-pointer fixture exported %d cases, expected 8", len(exports))
	}
	for _, selected := range exports {
		test.Run(selected.Name(), func(test *testing.T) {
			count := 1
			if selected.Name() == "Pair" || selected.Name() == "WrongPair" || selected.Name() == "NotTuple" {
				count = 2
			}
			raw := func() *types.Var { return types.NewVar(token.NoPos, nil, "", types.Typ[types.UnsafePointer]) }
			results := make([]*types.Var, count)
			for index := range results {
				results[index] = raw()
			}
			signature := types.NewSignatureType(nil, nil, nil, types.NewTuple(raw()), types.NewTuple(results...), false)
			access := gostdlib.AccessExport
			if selected.Name() == "StaticGood" || selected.Name() == "StaticWrong" {
				access = gostdlib.AccessStaticMethod
			}
			err := verifySourceRawPointers(resolvedConfig{providerRoot: root}, project, selected.Name(), signature, access,
				func(index int) (tsgo.ProjectTypeIdentity, error) {
					return project.CallableParameterTypeIdentity(selected, index)
				},
				func(index int) (tsgo.ProjectTypeIdentity, error) {
					return project.CallableResultTypeIdentity(selected, index, count)
				})
			valid := selected.Name() == "Good" || selected.Name() == "Pair" || selected.Name() == "StaticGood"
			if valid && err != nil || !valid && (err == nil || !strings.Contains(err.Error(), "verify source raw pointers")) {
				test.Fatalf("canonical=%v: %v", valid, err)
			}
		})
	}
}
