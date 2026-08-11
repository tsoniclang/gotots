package certify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderScalarContractRejectsMissingCertifiedAlias(t *testing.T) {
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
	scalarSource, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"src",
		"internal",
		"scalars.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(
		string(scalarSource),
		"export type int64 = bigint;\n",
		"",
		1,
	)
	if mutated == string(scalarSource) {
		t.Fatal("provider scalar mutation did not remove int64")
	}
	providerRoot := t.TempDir()
	scalarPath := filepath.Join(providerRoot, "src", "internal", "scalars.ts")
	if err := os.MkdirAll(filepath.Dir(scalarPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scalarPath, []byte(mutated), 0o644); err != nil {
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
	err = verifyProviderScalarContract(
		resolvedConfig{providerRoot: providerRoot},
		project,
		requirements,
	)
	if err == nil || !strings.Contains(err.Error(), "missing contract aliases") {
		t.Fatalf("missing provider scalar error = %v", err)
	}
}
