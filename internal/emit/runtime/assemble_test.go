package runtime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayRuntimeAssemblyExactJoinsRequestedDefinition(t *testing.T) {
	factory := tsgo.NewFactory()
	definitions, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 {
		t.Fatalf("array runtime definitions = %d, want one", len(definitions))
	}
	class, ok := definitions[0].Statement().(tsgo.ClassDeclaration)
	if !ok || class.Name().Text() != "GoArray" {
		t.Fatalf("array runtime statement = %T", definitions[0].Statement())
	}
}

func TestPointerHashIsAnOptionalExactRuntimeDefinition(t *testing.T) {
	base, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModulePointer,
		[]api.RuntimeSymbol{api.RuntimePointer},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 1 || base[0].Symbol() != api.RuntimePointer {
		t.Fatalf("base pointer definitions = %#v", base)
	}
	class, ok := base[0].Statement().(tsgo.ClassDeclaration)
	if !ok || len(class.Members()) != 19 {
		t.Fatalf("base pointer owner = %T with unexpected members", base[0].Statement())
	}

	withHash, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModulePointer,
		[]api.RuntimeSymbol{
			api.RuntimePointer,
			api.RuntimePointerHash,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(withHash) != 2 ||
		withHash[0].Symbol() != api.RuntimePointer ||
		withHash[1].Symbol() != api.RuntimePointerHash {
		t.Fatalf("pointer hash definitions = %#v", withHash)
	}
	if _, ok := withHash[1].Statement().(tsgo.FunctionDeclaration); !ok {
		t.Fatalf(
			"pointer hash definition = %T, want function",
			withHash[1].Statement(),
		)
	}
}

func TestAggregateArrayRuntimeAssemblyExactJoinsDemandedOperations(t *testing.T) {
	factory := tsgo.NewFactory()
	symbols := []api.RuntimeSymbol{
		api.RuntimeArray,
		api.RuntimeArrayAllocate,
		api.RuntimeArrayView,
		api.RuntimeArrayLocation,
	}
	definitions, err := Build(
		factory,
		api.RuntimeModuleArray,
		symbols,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != len(symbols) {
		t.Fatalf(
			"aggregate array definitions = %d, want %d",
			len(definitions),
			len(symbols),
		)
	}
	for index, definition := range definitions {
		if definition.Symbol() != symbols[index] {
			t.Fatalf(
				"aggregate definition %d = %d, want %d",
				index,
				definition.Symbol(),
				symbols[index],
			)
		}
		if index == 0 {
			if _, ok := definition.Statement().(tsgo.ClassDeclaration); !ok {
				t.Fatalf("array definition = %T, want class", definition.Statement())
			}
			continue
		}
		if _, ok := definition.Statement().(tsgo.FunctionDeclaration); !ok {
			t.Fatalf(
				"aggregate operation %d = %T, want function",
				index,
				definition.Statement(),
			)
		}
	}
}

func TestArrayRuntimeAssemblyRejectsMissingDuplicateAndWrongDefinitions(
	t *testing.T,
) {
	factory := tsgo.NewFactory()
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		nil,
	); err == nil {
		t.Fatal("missing RuntimeArray definition passed")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray, api.RuntimeArray},
	); err == nil {
		t.Fatal("duplicate RuntimeArray definition passed")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimePointer},
	); err == nil {
		t.Fatal("wrong-module runtime symbol passed array assembly")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{
			api.RuntimeArrayAllocate,
			api.RuntimeArray,
		},
	); err == nil {
		t.Fatal("aggregate operation preceding RuntimeArray passed assembly")
	}
}

func TestSliceAggregateDefinitionsExactJoinRequestedSymbols(t *testing.T) {
	symbols := []api.RuntimeSymbol{
		api.RuntimeSlice,
		api.RuntimeSliceAddress,
		api.RuntimeSliceStorage,
		api.RuntimeSliceArrayPointer,
		api.RuntimeArraySlice,
		api.RuntimeSliceAppendSlice,
		api.RuntimeSliceClear,
	}
	definitions, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleSlice,
		symbols,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != len(symbols) {
		t.Fatalf(
			"slice definition count = %d, want %d",
			len(definitions),
			len(symbols),
		)
	}
	for index, definition := range definitions {
		if definition.Symbol() != symbols[index] {
			t.Fatalf(
				"slice definition %d = %d, want %d",
				index,
				definition.Symbol(),
				symbols[index],
			)
		}
	}
}

func TestSliceAggregateDefinitionsRequireRuntimeSliceFirst(t *testing.T) {
	if _, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleSlice,
		[]api.RuntimeSymbol{api.RuntimeSliceStorage},
	); err == nil {
		t.Fatal("slice aggregate helper assembled without RuntimeSlice")
	}
}

func TestMapRuntimeRejectsDuplicateOwners(t *testing.T) {
	if _, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleMap,
		[]api.RuntimeSymbol{api.RuntimeMap, api.RuntimeMap},
	); err == nil {
		t.Fatal("map runtime accepted a duplicate owner")
	}
	definitions, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleMap,
		[]api.RuntimeSymbol{
			api.RuntimeMap,
			api.RuntimeMapValue,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 2 ||
		definitions[0].Symbol() != api.RuntimeMap ||
		definitions[1].Symbol() != api.RuntimeMapValue {
		t.Fatalf("map definitions = %#v", definitions)
	}
}
