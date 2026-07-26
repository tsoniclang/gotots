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
	definitions, err := runtimeemission.Build(
		tsgo.NewFactory(),
		api.RuntimeModuleSlice,
		[]api.RuntimeSymbol{api.RuntimeSlice},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 ||
		definitions[0].Symbol() != api.RuntimeSlice ||
		definitions[0].Statement().(tsgo.ClassDeclaration).Name().Text() !=
			api.RuntimeSliceExportName {
		t.Fatalf("runtime slice definitions = %#v, want exact frozen symbol", definitions)
	}
	if _, err := runtimeemission.Build(
		tsgo.NewFactory(),
		api.RuntimeModuleSlice,
		[]api.RuntimeSymbol{api.RuntimeSlice, api.RuntimeSlice},
	); err == nil {
		t.Fatal("runtime slice assembly accepted a non-exact symbol multiset")
	}
}

func TestRuntimeSliceOwnsOneClosedGenericDescriptor(t *testing.T) {
	class := runtimeslice.Build(tsgo.NewFactory())
	if class.Name().Text() != "RuntimeSlice" ||
		len(class.TypeParameters()) != 1 ||
		class.TypeParameters()[0].Name().Text() != "T" {
		t.Fatalf("runtime slice declaration = %#v", class)
	}
	members := class.Members()
	if len(members) != 10 {
		t.Fatalf("runtime slice members = %d, want constructor plus nine operations", len(members))
	}
	constructor, ok := members[0].(tsgo.ConstructorDeclaration)
	if !ok {
		t.Fatalf("runtime slice member 0 = %T, want constructor", members[0])
	}
	parameters := constructor.Parameters()
	if len(parameters) != 5 {
		t.Fatalf("runtime slice constructor parameters = %d, want five", len(parameters))
	}
	backing, ok := parameters[0].Type().(tsgo.UnionTypeNode)
	if !ok || len(backing.Types()) != 2 {
		t.Fatalf("runtime slice backing = %T, want T[] | null", parameters[0].Type())
	}
	if _, ok := backing.Types()[0].(tsgo.ArrayTypeNode); !ok {
		t.Fatalf("runtime slice internal backing = %T, want typed array", backing.Types()[0])
	}
	zeroName, ok := parameters[4].Name().(tsgo.Identifier)
	if !ok || zeroName.Text() != "zero" {
		t.Fatalf("runtime slice parameter 4 = %T, want typed zero", parameters[4].Name())
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
	reference, ok := source.(tsgo.TypeReferenceNode)
	if !ok {
		t.Fatalf("slice type = %T, want RuntimeSlice<T> reference", source)
	}
	name, ok := reference.TypeName().(tsgo.Identifier)
	if !ok || name.Text() != "RuntimeSlice" || len(reference.TypeArguments()) != 1 {
		t.Fatalf("slice reference = %#v, want RuntimeSlice with one concrete argument", reference)
	}
	if _, bare := source.(tsgo.ArrayTypeNode); bare {
		t.Fatal("source slice signature became a bare target array")
	}
}
