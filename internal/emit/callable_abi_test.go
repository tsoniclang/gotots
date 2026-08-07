package emit

import (
	"context"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAutomaticPointeeValueABIReconstructsDefinitionAndDirectCallers(
	t *testing.T,
) {
	target, _ := compileCallableABIFixture(t, "example.test/automatic", `package automatic

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

func ConditionalValue() int {
	current := 41
	return ReadThenCompute(&current)
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
		"export function Read(value: int): int {\n    return value;\n}",
		"export function ReadThenCompute(value: int): int",
		"return Read(current);",
		"return Read(GoPointer.dereference<int, int>(value).value);",
		"=> Read(GoPointer.dereference<int, int>",
		"Read(__gotots_argument_",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("automatic callable ABI lacks %q:\n%s", required, target)
		}
	}
	if strings.Contains(target, "GoPointer.cell") {
		t.Fatalf("automatic pointee-value call introduced a scalar cell:\n%s", target)
	}
}

func TestMutatingPointerMethodRetainsLocationABI(t *testing.T) {
	target, _ := compileCallableABIFixture(t, "example.test/mutatingmethod", `package mutatingmethod

type Counter struct{}

func (Counter) Increment(value *int) { (*value)++ }

func Apply(counter Counter, value *int) {
	counter.Increment(value)
}
`)
	for _, required := range []string{
		"Increment(value: GoPointer<int, int> | undefined): void",
		"const __gotots_store_0 = GoPointer.dereference<int, int>(value);",
		"__gotots_store_0.value = __gotots_store_0.value + 1;",
		"counter.Increment(value);",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("mutating pointer method lacks %q:\n%s", required, target)
		}
	}
}

func TestAutomaticPointeeValueABICoversMethodConsumers(t *testing.T) {
	target, program := compileCallableABIFixture(t, "example.test/methodabi", `package methodabi

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
	session, err := newProgramSession(program, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	reader := program.Roots()[0].Types().Scope().Lookup("Reader").Type()
	read := types.NewMethodSet(reader).Lookup(
		program.Roots()[0].Types(),
		"Read",
	).Obj().(*types.Func)
	if selected, ok := session.ResolveCallableABI(read.Origin()); !ok || !selected.Valid() {
		t.Fatal("projected method ABI is absent from the canonical artifact graph")
	}
	for _, required := range []string{
		"Read(value: int): int",
		"return receiver.Read(current);",
		"receiver.Read(GoPointer.dereference<int, int>(value).value)",
		"this.$go$value.Read(GoPointer.dereference<int, int>($argument0).value)",
		"goInterfaceNonNil<Contract>(__gotots_receiver_2).Read(__gotots_argument_3)",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("method callable ABI lacks %q:\n%s", required, target)
		}
	}
	if strings.Contains(target, "Read(GoPointer.cell") {
		t.Fatalf("projected method call introduced a scalar cell:\n%s", target)
	}
}

func compileCallableABIFixture(
	t *testing.T,
	module string,
	sourceText string,
) (string, *load.Program) {
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
	return source.String(), program
}
