package pointer_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	"github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildCreatesOneTypedCanonicalLocationClass(t *testing.T) {
	factory := tsgo.NewFactory()
	statement := pointer.Build(
		factory,
		pointerClassName(t),
		panicClassName(t),
	)

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
	if len(parameters) != 2 ||
		parameters[0].Name().Text() != "L" ||
		parameters[1].Name().Text() != "S" {
		t.Fatalf("pointer type parameters = %v, want L and S", parameters)
	}
	members := class.Members()
	if len(members) != 19 {
		t.Fatalf("pointer class members = %d, want 19", len(members))
	}
	constructor, ok := members[3].(tsgo.ConstructorDeclaration)
	if !ok {
		t.Fatalf("pointer member = %T, want constructor", members[3])
	}
	constructorParameters := constructor.Parameters()
	if len(constructorParameters) != 3 {
		t.Fatalf("pointer constructor parameters = %d, want 3", len(constructorParameters))
	}
	for index, name := range []string{pointer.AddressName, "read", "write"} {
		parameter := constructorParameters[index]
		modifiers := parameter.Modifiers()
		validModifiers := len(modifiers) == 2 &&
			modifiers[0].Kind() == tsgo.SyntaxKindPrivateKeyword &&
			modifiers[1].Kind() == tsgo.SyntaxKindReadonlyKeyword
		if index == 0 {
			validModifiers = len(modifiers) == 1 &&
				modifiers[0].Kind() == tsgo.SyntaxKindReadonlyKeyword
		}
		if parameter.Name().(tsgo.Identifier).Text() != name ||
			!validModifiers {
			t.Fatalf("pointer constructor parameter %d = %#v", index, parameter)
		}
	}
	guard := pointerMethod(t, class, pointer.DereferenceName)
	if guard.Name().(tsgo.Identifier).Text() != pointer.DereferenceName ||
		len(guard.Modifiers()) != 1 ||
		guard.Modifiers()[0].Kind() != tsgo.SyntaxKindStaticKeyword ||
		len(guard.TypeParameters()) != 2 ||
		len(guard.Parameters()) != 1 {
		t.Fatalf("pointer guard = %T, want static typed dereference", guard)
	}
}

func TestNilDereferenceSuccessMutationRemovesRequiredThrow(t *testing.T) {
	class := pointer.Build(
		tsgo.NewFactory(),
		pointerClassName(t),
		panicClassName(t),
	).(tsgo.ClassDeclaration)
	guard := pointerMethod(t, class, pointer.DereferenceName)
	body := guard.Body().(tsgo.Block).Statements()
	if len(body) != 2 {
		t.Fatalf("pointer guard statements = %d, want check and return", len(body))
	}
	condition, ok := body[0].(tsgo.IfStatement)
	if !ok {
		t.Fatalf("pointer guard first statement = %T, want IfStatement", body[0])
	}
	failure, ok := condition.ThenStatement().(tsgo.Block)
	if !ok || len(failure.Statements()) != 1 {
		t.Fatal("nil pointer branch has no failure")
	}
	call, ok := failure.Statements()[0].(tsgo.ExpressionStatement).
		Expression().(tsgo.CallExpression)
	if !ok ||
		call.Expression().(tsgo.PropertyAccessExpression).
			Expression().(tsgo.Identifier).Text() != panicClassName(t) {
		t.Fatal("nil pointer branch bypasses the shared panic ABI")
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

func TestBuildPrintsSourceShapedCanonicalLocations(t *testing.T) {
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
		pointer.Build(
			factory,
			pointerClassName(t),
			panicClassName(t),
		),
		tsgo.PrintOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"export class GoPointer<L, S>",
		"private static readonly roots: WeakMap<object, object>",
		"private constructor(readonly $go$address: object",
		"static cell<L, S>(value: S): GoPointer<L, S>",
		"static field<L, PL, PS extends object, K extends keyof PS>",
		"static objectField<L, O extends object, K extends keyof O>",
		"static elementView<L, S, O>",
		"static index<L, S, PL, O extends",
		"static indexView<L, S, PL, V, O extends",
		"static arrayRegion<L, T, S extends",
		"const numericIndex = Number(index);",
		"static equal<LL, LS, RL, RS>",
		"static dereference<L, S>",
		"static view<F, T, S>",
		"get value(): S",
		"set value(value: S)",
		`GoPanic.raiseRuntime("nil pointer dereference")`,
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("pointer runtime lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{"any", "unknown", ".call(", ".apply(", ".bind("} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("pointer runtime contains %q:\n%s", forbidden, printed)
		}
	}
}

func pointerMethod(
	t *testing.T,
	class tsgo.ClassDeclaration,
	name string,
) tsgo.MethodDeclaration {
	t.Helper()
	for _, member := range class.Members() {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			continue
		}
		identifier, ok := method.Name().(tsgo.Identifier)
		if ok && identifier.Text() == name {
			return method
		}
	}
	t.Fatalf("pointer method %q is absent", name)
	return nil
}

func pointerClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimePointer)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}

func panicClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}
