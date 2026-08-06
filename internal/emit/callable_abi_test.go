package emit

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAutomaticPointeeValueABIReconstructsDefinitionAndDirectCallers(
	t *testing.T,
) {
	root := t.TempDir()
	writeSourceImplementationFixture(
		t,
		filepath.Join(root, "go.mod"),
		"module example.test/automatic\n\ngo 1.26.4\n",
	)
	writeSourceImplementationFixture(t, filepath.Join(root, "automatic.go"), `package automatic

func Read(value *int) int { return *value }

func Value() int {
	current := 41
	return Read(&current)
}

func Existing(value *int) int { return Read(value) }

func ThroughValue(value *int) int {
	read := Read
	return read(value)
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: root,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repository, root)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	var source strings.Builder
	for _, file := range emission.Files() {
		printed, printErr := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if printErr != nil {
			t.Fatal(printErr)
		}
		if strings.Contains(printed, "function Read") ||
			strings.Contains(printed, "function Value") ||
			strings.Contains(printed, "function Existing") ||
			strings.Contains(printed, "function ThroughValue") {
			source.WriteString(printed)
		}
	}
	target := source.String()
	for _, required := range []string{
		"export function Read(value: int): int {\n    return value;\n}",
		"return Read(current);",
		"return Read(GoPointer.dereference<int, int>(value).value);",
		"=> Read(GoPointer.dereference<int, int>",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("automatic callable ABI lacks %q:\n%s", required, target)
		}
	}
	if strings.Contains(target, "GoPointer.cell") {
		t.Fatalf("automatic pointee-value call introduced a scalar cell:\n%s", target)
	}
}
