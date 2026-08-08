package emit

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCanonicalPointerCallablePreservesDefinitionAndDirectCallers(
	t *testing.T,
) {
	target := compilePointerCallableFixture(t, "example.test/pointercall", `package pointercall

func Read(value *int) int { return *value }

func ReadThenCompute(value *int) int {
	current := *value
	current++
	return current + *value
}

func Value() int {
	current := 41
	return Read(&current)
}

func Existing(value *int) int { return Read(value) }

func ThroughValue(value *int) int {
	read := Read
	return read(value)
}

func Deferred(value *int) {
	defer Read(value)
}
`)
	for _, required := range []string{
		"export function Read(value: Pointer<int> | undefined): int",
		"return loadPointer<int>((value ?? GoPanic.raiseRuntime",
		"export function ReadThenCompute(value: Pointer<int> | undefined): int",
		"return Read(addressOf<int>(current));",
		"return Read(value);",
		"(($0: Pointer<int> | undefined) => int)",
		"Read(__gotots_argument_",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("canonical pointer callable lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{"GoPointer", "runtime/pointer", "Read(value: int): int"} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("canonical pointer callable contains obsolete route %q:\n%s", forbidden, target)
		}
	}
}

func TestMutatingPointerMethodPreservesLocationContract(t *testing.T) {
	target := compilePointerCallableFixture(t, "example.test/mutatingmethod", `package mutatingmethod

type Counter struct{}

func (Counter) Increment(value *int) { (*value)++ }

func Apply(counter Counter, value *int) {
	counter.Increment(value)
}
	`)
	for _, required := range []string{
		"Increment(value: Pointer<int> | undefined): void",
		"const __gotots_store_0 = (value ?? GoPanic.raiseRuntime",
		"storePointer(__gotots_store_0, loadPointer(__gotots_store_0) + 1);",
		"counter.Increment(value);",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("mutating pointer method lacks %q:\n%s", required, target)
		}
	}
}

func TestCanonicalPointerCallableCoversMethodConsumers(t *testing.T) {
	target := compilePointerCallableFixture(t, "example.test/methodcall", `package methodcall

type Reader struct{}

func (Reader) Read(value *int) int { return *value }

func Direct(receiver Reader) int {
	current := 41
	return receiver.Read(&current)
}

func Existing(receiver Reader, value *int) int {
	return receiver.Read(value)
}

func ThroughValue(receiver Reader, value *int) int {
	read := receiver.Read
	return read(value)
}

func ThroughExpression(value *int) int {
	read := Reader.Read
	return read(Reader{}, value)
}

func Deferred(receiver Reader, value *int) {
	defer receiver.Read(value)
}

type Contract interface {
	Read(value *int) int
}

func ThroughInterface(value *int) int {
	var contract Contract = Reader{}
	return contract.Read(value)
}
`)
	for _, required := range []string{
		"Read(value: Pointer<int> | undefined): int",
		"return receiver.Read(addressOf<int>(current));",
		"return receiver.Read(value);",
		"this.$go$value.Read($argument0)",
		"goInterfaceNonNil<Contract>(__gotots_receiver_2).Read(__gotots_argument_3)",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("pointer method callable lacks %q:\n%s", required, target)
		}
	}
	if strings.Contains(target, "GoPointer") || strings.Contains(target, "runtime/pointer") {
		t.Fatalf("canonical pointer method retained the obsolete pointer runtime:\n%s", target)
	}
}

func compilePointerCallableFixture(
	t *testing.T,
	module string,
	sourceText string,
) string {
	t.Helper()
	root := t.TempDir()
	writeSourceImplementationFixture(
		t,
		filepath.Join(root, "go.mod"),
		"module "+module+"\n\ngo 1.26.4\n",
	)
	writeSourceImplementationFixture(
		t,
		filepath.Join(root, "fixture.go"),
		sourceText,
	)
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
		source.WriteString(printed)
	}
	return source.String()
}
