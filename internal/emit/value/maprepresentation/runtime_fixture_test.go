package maprepresentation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func materializeSpecializationRuntime(t *testing.T, directory string, symbols []api.RuntimeSymbol) {
	t.Helper()
	abi, err := api.NewScalarABI(api.IntegerRepresentationNumber, api.NativeIntegerWidth64)
	if err != nil {
		t.Fatal(err)
	}
	requested := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		requested[symbol] = struct{}{}
	}
	assembled, err := runtimeemission.AssemblePackage(tsgo.NewFactory(), abi, requested, nil)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(specializationRepositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	for _, file := range assembled.Files() {
		source, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, file.OutputPath())
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte("{\"type\":\"module\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runtimefixture.InstallResolution(directory, filepath.Join(directory, "out")); err != nil {
		t.Fatal(err)
	}
}
