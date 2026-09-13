package complex_test

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestComplexPhysicalStorageExecutesDifferentially(test *testing.T) {
	fixture := filepath.Join(repositoryRoot(), "testdata", "constructs", "value", "complexstorage")
	loaded, err := load.One(context.Background(), load.Request{Directory: fixture, Pattern: "."})
	if err != nil {
		test.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		test.Fatal(err)
	}
	directory := test.TempDir()
	paths, module, printed := printComplex(test, directory, emission)
	for _, required := range []string{"Small: field<typeof GoComplex64Storage>()", "Large: field<typeof GoComplex128Storage>()", "goComplex64ToStorage", "goComplex128FromStorage"} {
		if !strings.Contains(printed, required) {
			test.Fatalf("storage differential did not exercise %q", required)
		}
	}
	goRunner := filepath.Join(directory, "native")
	writeFile(test, filepath.Join(goRunner, "go.mod"), "module example.com/runner\n\ngo 1.26.4\n\nrequire example.com/complexvalues v0.0.0\nreplace example.com/complexvalues => "+filepath.ToSlash(fixture)+"\n")
	writeFile(test, filepath.Join(goRunner, "main.go"), `package main
import (
    "fmt"
    values "example.com/complexvalues"
)
func main() {
    first := values.StorageCopies()
    second := values.StorageContainers()
    fmt.Println(real(first), imag(first), real(second), imag(second))
}
`)
	native := runCommand(test, goRunner, filepath.Join(runtime.GOROOT(), "bin", "go"), "run", ".")
	actual := runComplexScript(test, directory, paths, `import * as values from "`+module+`";
const first = values.StorageCopies();
const second = values.StorageContainers();
console.log(first.real, first.imag, second.real, second.imag);
`)
	if actual != native {
		test.Fatalf("complex storage differs: target=%q native=%q", actual, native)
	}
}
