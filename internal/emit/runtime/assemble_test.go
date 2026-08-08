package runtime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	channelruntime "github.com/tsoniclang/gotots/internal/emit/runtime/channel"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArrayRuntimeAssemblyExactJoinsRequestedDefinition(t *testing.T) {
	factory := tsgo.NewFactory()
	definitions, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray},
		api.ConcurrencySemanticsDisabled,
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

func TestErrorRuntimeContractFollowsConcurrencyProfile(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		concurrency api.ConcurrencySemantics
		awaitable   bool
	}{
		{"disabled", api.ConcurrencySemanticsDisabled, false},
		{"cooperative", api.ConcurrencySemanticsCooperative, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			definitions, err := Build(
				tsgo.NewFactory(),
				api.RuntimeModuleInterfaceValue,
				[]api.RuntimeSymbol{
					api.RuntimeInterfaceValue,
					api.RuntimeBuiltinErrorType,
				},
				testCase.concurrency,
			)
			if err != nil {
				t.Fatal(err)
			}
			contract := definitions[1].Statement().(tsgo.InterfaceDeclaration)
			method := contract.Members()[0].(tsgo.MethodSignatureDeclaration)
			reference, isReference := method.Type().(tsgo.TypeReferenceNode)
			if isReference != testCase.awaitable {
				t.Fatalf("Error result = %T, awaitable=%t", method.Type(), testCase.awaitable)
			}
			if isReference && reference.TypeName().(tsgo.Identifier).Text() != "Awaitable" {
				t.Fatalf("cooperative Error result = %#v", reference)
			}
		})
	}
}

func TestEmptyStructRuntimeHasOneExactNominalOwner(t *testing.T) {
	definitions, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleStruct,
		[]api.RuntimeSymbol{api.RuntimeEmptyStruct},
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 ||
		definitions[0].Symbol() != api.RuntimeEmptyStruct {
		t.Fatalf("empty-struct definitions = %#v", definitions)
	}
	class, ok := definitions[0].Statement().(tsgo.ClassDeclaration)
	if !ok || class.Name().Text() != "GoEmptyStruct" || len(class.Members()) != 10 {
		t.Fatalf(
			"empty-struct owner = %T with unexpected shape",
			definitions[0].Statement(),
		)
	}
}

func TestUnsafeRuntimeExactJoinsStringIntrinsicDefinition(t *testing.T) {
	symbols := []api.RuntimeSymbol{
		api.RuntimeUnsafeString,
	}
	definitions, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleUnsafe,
		symbols,
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != len(symbols) {
		t.Fatalf("unsafe runtime definitions = %d, want %d", len(definitions), len(symbols))
	}
	for index, definition := range definitions {
		if definition.Symbol() != symbols[index] {
			t.Fatalf("unsafe definition %d = %d, want %d", index, definition.Symbol(), symbols[index])
		}
		if _, ok := definition.Statement().(tsgo.FunctionDeclaration); !ok {
			t.Fatalf("unsafe definition %d = %T, want function", index, definition.Statement())
		}
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
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
	); err == nil {
		t.Fatal("missing RuntimeArray definition passed")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeArray, api.RuntimeArray},
		api.ConcurrencySemanticsDisabled,
	); err == nil {
		t.Fatal("duplicate RuntimeArray definition passed")
	}
	if _, err := Build(
		factory,
		api.RuntimeModuleArray,
		[]api.RuntimeSymbol{api.RuntimeStringIndex},
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
	); err == nil {
		t.Fatal("slice aggregate helper assembled without RuntimeSlice")
	}
}

func TestMapRuntimeRejectsDuplicateOwners(t *testing.T) {
	if _, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModuleMap,
		[]api.RuntimeSymbol{api.RuntimeMap, api.RuntimeMap},
		api.ConcurrencySemanticsDisabled,
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
		api.ConcurrencySemanticsDisabled,
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

func TestPanicRuntimeCreationStaysOnTheCanonicalCarrier(t *testing.T) {
	definitions, err := Build(
		tsgo.NewFactory(),
		api.RuntimeModulePanic,
		[]api.RuntimeSymbol{api.RuntimePanic},
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 {
		t.Fatalf("panic definitions = %d, want 1", len(definitions))
	}
	carrier := definitions[0].Statement().(tsgo.ClassDeclaration)
	constructor := carrier.Members()[0].(tsgo.ConstructorDeclaration)
	create := carrier.Members()[1].(tsgo.MethodDeclaration)
	if constructor.Modifiers()[0].Kind() != tsgo.SyntaxKindPrivateKeyword ||
		create.Name().(tsgo.Identifier).Text() != "createRuntime" {
		t.Fatal("runtime panic creation escaped the canonical panic carrier")
	}
}

func TestSchedulerRoutesOnlyThroughItsSemanticOwner(t *testing.T) {
	factory := tsgo.NewFactory()
	if _, err := channelruntime.Build(
		factory,
		api.RuntimeScheduler,
		"GoChannel",
		"GoReceiveChannel",
		"GoSendChannel",
		"GoSelectCase",
		"goSelect",
		"goSelectReady",
		"goSelectAttempt",
		"GoPanic",
		"GoDenseIndex",
	); err == nil {
		t.Fatal("channel owner accepted RuntimeScheduler")
	}
	definitions, err := Build(
		factory,
		api.RuntimeModuleChannel,
		[]api.RuntimeSymbol{api.RuntimeScheduler},
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 1 ||
		definitions[0].Symbol() != api.RuntimeScheduler {
		t.Fatalf("scheduler definitions = %#v", definitions)
	}
	class, ok := definitions[0].Statement().(tsgo.ClassDeclaration)
	if !ok || class.Name().Text() != "GoScheduler" {
		t.Fatalf(
			"scheduler semantic owner produced %T",
			definitions[0].Statement(),
		)
	}
}
