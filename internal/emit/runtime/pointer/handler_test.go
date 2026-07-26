package pointer_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesOneTypedGenericCellClass(t *testing.T) {
	factory := tsgo.NewFactory()
	statement := pointer.Build(factory, pointerClassName(t))

	class, ok := statement.(tsgo.ClassDeclaration)
	if !ok {
		t.Fatalf("pointer runtime = %T, want tsgo.ClassDeclaration", statement)
	}
	if class.Name().Text() != pointerClassName(t) {
		t.Fatalf("pointer class name = %q, want GoPointer", class.Name().Text())
	}
	if modifiers := class.Modifiers(); len(modifiers) != 1 ||
		modifiers[0].Kind() != tsgo.SyntaxKindExportKeyword {
		t.Fatalf("pointer class modifiers = %v, want export", modifiers)
	}
	parameters := class.TypeParameters()
	if len(parameters) != 1 || parameters[0].Name().Text() != "T" {
		t.Fatalf("pointer type parameters = %v, want T", parameters)
	}
	members := class.Members()
	if len(members) != 2 {
		t.Fatalf("pointer class members = %d, want constructor and guard", len(members))
	}
	constructor, ok := members[0].(tsgo.ConstructorDeclaration)
	if !ok {
		t.Fatalf("pointer member = %T, want constructor", members[0])
	}
	cell := constructor.Parameters()
	if len(cell) != 1 ||
		cell[0].Name().(tsgo.Identifier).Text() != pointer.CellValueName ||
		len(cell[0].Modifiers()) != 1 ||
		cell[0].Modifiers()[0].Kind() != tsgo.SyntaxKindPublicKeyword {
		t.Fatalf("pointer constructor parameter is not public value: %v", cell)
	}
	cellType, ok := cell[0].Type().(tsgo.TypeReferenceNode)
	if !ok || cellType.TypeName().(tsgo.Identifier).Text() != "T" {
		t.Fatalf("pointer cell type = %T, want T", cell[0].Type())
	}
	guard, ok := members[1].(tsgo.MethodDeclaration)
	if !ok ||
		guard.Name().(tsgo.Identifier).Text() != pointer.DereferenceName ||
		len(guard.Modifiers()) != 1 ||
		guard.Modifiers()[0].Kind() != tsgo.SyntaxKindStaticKeyword ||
		len(guard.TypeParameters()) != 1 ||
		len(guard.Parameters()) != 1 {
		t.Fatalf("pointer guard = %T, want static typed dereference", members[1])
	}
}

func TestNilDereferenceSuccessMutationRemovesRequiredThrow(t *testing.T) {
	class := pointer.Build(
		tsgo.NewFactory(),
		pointerClassName(t),
	).(tsgo.ClassDeclaration)
	guard := class.Members()[1].(tsgo.MethodDeclaration)
	body := guard.Body().(tsgo.Block).Statements()
	if len(body) != 2 {
		t.Fatalf("pointer guard statements = %d, want check and return", len(body))
	}
	condition, ok := body[0].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("pointer guard first statement = %T, want IfStatement", body[0])
	}
	failure, ok := condition.ThenStatement().(tsgo.Block)
	if !ok ||
		len(failure.Statements()) != 1 ||
		failure.Statements()[0].Kind() != tsgo.SyntaxKindThrowStatement {
		t.Fatal("nil pointer branch does not throw")
	}
}

func TestPointerRuntimeBuildExactJoinsItsFrozenSymbol(t *testing.T) {
	factory := tsgo.NewFactory()
	definitions, err := runtimeemission.Build(
		factory,
		api.RuntimeModulePointer,
		[]api.RuntimeSymbol{api.RuntimePointer},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 ||
		definitions[0].Symbol() != api.RuntimePointer {
		t.Fatalf("pointer definitions = %#v, want one RuntimePointer", definitions)
	}
	for _, symbols := range [][]api.RuntimeSymbol{
		{api.RuntimePointer, api.RuntimePointer},
		{api.RuntimeStringIndex},
	} {
		_, err := runtimeemission.Build(
			factory,
			api.RuntimeModulePointer,
			symbols,
		)
		var assemblyError *runtimeemission.AssemblyError
		if !errors.As(err, &assemblyError) {
			t.Fatalf("symbols %v error = %v, want AssemblyError", symbols, err)
		}
	}
}

func TestBuildPrintsSourceShapedCell(t *testing.T) {
	factory := tsgo.NewFactory()
	client, err := tsgo.StartClient(
		filepath.Join("..", "..", "..", ".."),
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	printed, err := client.PrintNode(
		pointer.Build(factory, pointerClassName(t)),
		tsgo.PrintOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `export class GoPointer<T> {
    constructor(public value: T) {
    }
    static dereference<T>(pointer: GoPointer<T> | undefined): GoPointer<T> {
        if (pointer === void 0) {
            throw new Error("nil pointer dereference");
        }
        return pointer;
    }
}`
	if printed != expected {
		t.Fatalf("pointer runtime:\n%s\nwant:\n%s", printed, expected)
	}
}

func pointerClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimePointer)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}
