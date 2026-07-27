package stringruntime_test

import (
	"errors"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	stringruntime "github.com/tsoniclang/gotots/internal/emit/runtime/string"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBuildExactJoinsRequestedStringSymbols(t *testing.T) {
	definitions, err := runtimeemission.Build(
		tsgo.NewFactory(),
		api.RuntimeModuleString,
		[]api.RuntimeSymbol{
			api.RuntimeStringIndex,
			api.RuntimeStringSlice,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 2 ||
		definitions[0].Symbol() != api.RuntimeStringIndex ||
		definitions[1].Symbol() != api.RuntimeStringSlice {
		t.Fatalf("runtime definitions = %#v", definitions)
	}
	index, indexOK := definitions[0].Statement().(tsgo.FunctionDeclaration)
	slice, sliceOK := definitions[1].Statement().(tsgo.FunctionDeclaration)
	if !indexOK ||
		index.Name().Text() != "goStringIndex" ||
		!sliceOK ||
		slice.Name().Text() != "goStringSlice" {
		t.Fatalf("runtime statements = %T/%T", definitions[0].Statement(), definitions[1].Statement())
	}
}

func TestBuildEmitsOnlyTheDemandedStringDefinition(t *testing.T) {
	for _, symbol := range []api.RuntimeSymbol{
		api.RuntimeStringIndex,
		api.RuntimeStringSlice,
	} {
		t.Run(apiName(t, symbol), func(t *testing.T) {
			definitions, err := runtimeemission.Build(
				tsgo.NewFactory(),
				api.RuntimeModuleString,
				[]api.RuntimeSymbol{symbol},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(definitions) != 1 || definitions[0].Symbol() != symbol {
				t.Fatalf("runtime definitions = %#v, want only symbol %d", definitions, symbol)
			}
			function, ok := definitions[0].Statement().(tsgo.FunctionDeclaration)
			if !ok || function.Name().Text() != apiName(t, symbol) {
				t.Fatalf("runtime definition = %T, want %s", definitions[0].Statement(), apiName(t, symbol))
			}
		})
	}
}

func TestBuildRejectsNonStringSymbol(t *testing.T) {
	_, err := stringruntime.Build(
		tsgo.NewFactory(),
		[]api.RuntimeSymbol{api.RuntimePointer},
		apiName(t, api.RuntimePanic),
	)
	var buildError *stringruntime.BuildError
	if !errors.As(err, &buildError) || buildError.Symbol != api.RuntimePointer {
		t.Fatalf("error = %#v, want string runtime build error", err)
	}
}

func apiName(t *testing.T, symbol api.RuntimeSymbol) string {
	t.Helper()
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		t.Fatal(err)
	}
	return contract.ExportedName()
}
