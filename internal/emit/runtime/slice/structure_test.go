package slice_test

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRuntimeAssemblyExactJoinsFrozenSliceSymbol(t *testing.T) {
	className := runtimeSliceClassName(t)
	definitions, err := runtimeemission.Build(
		tsgo.NewFactory(),
		api.RuntimeModuleSlice,
		[]api.RuntimeSymbol{api.RuntimeSlice},
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 ||
		definitions[0].Symbol() != api.RuntimeSlice ||
		definitions[0].Statement().(tsgo.ClassDeclaration).Name().Text() !=
			className {
		t.Fatalf("runtime slice definitions = %#v, want exact frozen symbol", definitions)
	}
	if _, err := runtimeemission.Build(
		tsgo.NewFactory(),
		api.RuntimeModuleSlice,
		[]api.RuntimeSymbol{api.RuntimeSlice, api.RuntimeSlice},
		api.ConcurrencySemanticsDisabled,
	); err == nil {
		t.Fatal("runtime slice assembly accepted a non-exact symbol multiset")
	}
}

func TestRuntimeSliceBuilderConsumesInjectedContractName(t *testing.T) {
	const changedContractName = "ChangedRuntimeSlice"
	class := runtimeslice.Build(
		tsgo.NewFactory(),
		changedContractName,
		runtimePanicClassName(t),
		runtimeDenseIndexClassName(t),
	)
	if class.Name().Text() != changedContractName {
		t.Fatalf(
			"runtime slice declaration = %q, want injected contract name",
			class.Name().Text(),
		)
	}
	nilMethod := class.Members()[1].(tsgo.MethodDeclaration)
	assertTypeReferenceName(t, nilMethod.Type(), changedContractName)
	constructor := nilMethod.Body().(tsgo.Block).
		Statements()[0].(tsgo.ReturnStatement).
		Expression().(tsgo.NewExpression).
		Expression().(tsgo.Identifier)
	if constructor.Text() != changedContractName {
		t.Fatalf(
			"runtime slice self-construction = %q, want injected contract name",
			constructor.Text(),
		)
	}
}

func TestRuntimeSliceOwnsOneClosedGenericDescriptor(t *testing.T) {
	className := runtimeSliceClassName(t)
	class := runtimeslice.Build(
		tsgo.NewFactory(),
		className,
		runtimePanicClassName(t),
		runtimeDenseIndexClassName(t),
	)
	if class.Name().Text() != className ||
		len(class.TypeParameters()) != 1 ||
		class.TypeParameters()[0].Name().Text() != "T" {
		t.Fatalf("runtime slice declaration = %#v", class)
	}
	members := class.Members()
	if len(members) != 10 {
		t.Fatalf("runtime slice members = %d, want constructor plus nine core operations", len(members))
	}
	constructor, ok := members[0].(tsgo.ConstructorDeclaration)
	if !ok {
		t.Fatalf("runtime slice member 0 = %T, want constructor", members[0])
	}
	parameters := constructor.Parameters()
	if len(parameters) != 4 {
		t.Fatalf("runtime slice constructor parameters = %d, want four", len(parameters))
	}
	backing, ok := parameters[0].Type().(tsgo.UnionTypeNode)
	if !ok || len(backing.Types()) != 2 {
		t.Fatalf("runtime slice backing = %T, want T[] | null", parameters[0].Type())
	}
	if _, ok := backing.Types()[0].(tsgo.ArrayTypeNode); !ok {
		t.Fatalf("runtime slice internal backing = %T, want typed array", backing.Types()[0])
	}
	var methods []string
	for _, member := range members[1:] {
		method, ok := member.(tsgo.MethodDeclaration)
		if !ok {
			t.Fatalf("runtime slice member = %T, want method", member)
		}
		methods = append(methods, method.Name().(tsgo.Identifier).Text())
	}
	want := []string{
		"nil",
		"make",
		"literal",
		"isNil",
		"get",
		"set",
		"slice",
		"append",
		"copy",
	}
	if !slices.Equal(methods, want) {
		t.Fatalf("runtime slice methods = %v, want %v", methods, want)
	}
}

func TestSourceSliceSignaturesUseRuntimeDescriptorNotBareArray(t *testing.T) {
	emission := compileFixture(t)
	var identity tsgo.FunctionDeclaration
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == "Identity" {
				identity = function
			}
		}
	}
	if identity == nil {
		t.Fatal("Identity target function is absent")
	}
	if len(identity.Parameters()) != 1 {
		t.Fatalf("Identity parameters = %d, want one", len(identity.Parameters()))
	}
	assertRuntimeSliceType(t, identity.Parameters()[0].Type())
	assertRuntimeSliceType(t, identity.Type())
}

func assertRuntimeSliceType(t *testing.T, source tsgo.TypeNode) {
	t.Helper()
	assertTypeReferenceName(t, source, runtimeSliceClassName(t))
}

func assertTypeReferenceName(
	t *testing.T,
	source tsgo.TypeNode,
	expected string,
) {
	t.Helper()
	reference, ok := source.(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("slice type = %T, want RuntimeSlice<T> reference", source)
	}
	name, ok := reference.TypeName().(tsgo.Identifier)
	if !ok || name.Text() != expected || len(reference.TypeArguments()) != 1 {
		t.Fatalf("slice reference = %#v, want RuntimeSlice with one concrete argument", reference)
	}
	if _, bare := source.(tsgo.ArrayTypeNode); bare {
		t.Fatal("source slice signature became a bare target array")
	}
}

func runtimeSliceClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimeSlice)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}

func runtimePanicClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimePanic)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}

func runtimeDenseIndexClassName(t *testing.T) string {
	t.Helper()
	contract, err := api.RuntimeContract(api.RuntimeDenseIndex)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}
