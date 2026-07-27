package api_test

import (
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRuntimeSymbolContractsArePinnedAndClosed(t *testing.T) {
	tests := []struct {
		symbol api.RuntimeSymbol
		id     uint16
		module api.RuntimeModule
		path   string
		name   string
		typeOK bool
		deps   []api.RuntimeSymbol
	}{
		{api.RuntimeStringIndex, 1, api.RuntimeModuleString, "runtime/string.ts", "goStringIndex", false, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimeStringSlice, 2, api.RuntimeModuleString, "runtime/string.ts", "goStringSlice", false, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimePointer, 100, api.RuntimeModulePointer, "runtime/pointer.ts", "GoPointer", true, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimeArray, 200, api.RuntimeModuleArray, "runtime/array.ts", "GoArray", true, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimeSlice, 300, api.RuntimeModuleSlice, "runtime/slice.ts", "RuntimeSlice", true, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimeSliceAddress, 301, api.RuntimeModuleSlice, "runtime/slice.ts", "goSliceAddress", false, []api.RuntimeSymbol{api.RuntimeSlice, api.RuntimePointer}},
		{api.RuntimeMap, 400, api.RuntimeModuleMap, "runtime/map.ts", "GoMap", true, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimePanic, 500, api.RuntimeModulePanic, "runtime/panic.ts", "GoPanic", true, nil},
		{api.RuntimeIntegerDivide, 600, api.RuntimeModuleInteger, "runtime/integer.ts", "goIntegerDivide", false, []api.RuntimeSymbol{api.RuntimePanic}},
		{api.RuntimeIntegerRemainder, 601, api.RuntimeModuleInteger, "runtime/integer.ts", "goIntegerRemainder", false, []api.RuntimeSymbol{api.RuntimePanic}},
	}
	for _, test := range tests {
		if uint16(test.symbol) != test.id {
			t.Fatalf("runtime symbol %v id = %d, want %d", test.symbol, test.symbol, test.id)
		}
		contract, err := api.RuntimeContract(test.symbol)
		if err != nil {
			t.Fatalf("contract for %v: %v", test.symbol, err)
		}
		if contract.Module() != test.module ||
			contract.OutputPath() != test.path ||
			contract.ExportedName() != test.name ||
			!slices.Equal(contract.Dependencies(), test.deps) ||
			contract.AllowsImportPhase(api.ImportPhaseType) != test.typeOK ||
			!contract.AllowsImportPhase(api.ImportPhaseValue) {
			t.Fatalf(
				"contract for %v = (%v, %q, %q, type=%v, value=%v)",
				test.symbol,
				contract.Module(),
				contract.OutputPath(),
				contract.ExportedName(),
				contract.AllowsImportPhase(api.ImportPhaseType),
				contract.AllowsImportPhase(api.ImportPhaseValue),
			)
		}
	}
	if _, err := api.RuntimeContract(api.RuntimeInvalid); err == nil {
		t.Fatal("invalid runtime symbol unexpectedly has a contract")
	}
}

func TestRuntimeImportRequestCarriesDefinitionIdentity(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewRuntimeImportRequest(
		factory,
		api.ImportPhaseValue,
		"../../../runtime/string.js",
		api.RuntimeStringIndex,
		"index__runtime",
	)
	if err != nil {
		t.Fatal(err)
	}
	if request.Kind() != api.RootRequestImport ||
		request.ImportPhase() != api.ImportPhaseValue ||
		request.ModulePath() != "../../../runtime/string.js" ||
		request.ExportedName() != "goStringIndex" ||
		request.LocalName() != "index__runtime" {
		t.Fatalf("runtime import request = %#v", request)
	}
	symbol, ok := request.RuntimeSymbol()
	if !ok || symbol != api.RuntimeStringIndex {
		t.Fatalf("runtime symbol = %v, %v", symbol, ok)
	}
}

func TestRuntimeContractDoesNotExposeDependencyBacking(t *testing.T) {
	contract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		t.Fatal(err)
	}
	dependencies := contract.Dependencies()
	dependencies[0] = api.RuntimeInvalid
	if actual := contract.Dependencies(); len(actual) != 1 ||
		actual[0] != api.RuntimePanic {
		t.Fatalf("runtime dependencies leaked mutable backing: %v", actual)
	}
}

func TestRuntimeClassRequestAllowsTypeUseAndRejectsHelperTypeUse(t *testing.T) {
	factory := tsgo.NewFactory()
	request, err := api.NewRuntimeImportRequest(
		factory,
		api.ImportPhaseType,
		"../../../runtime/pointer.js",
		api.RuntimePointer,
		"PointerType",
	)
	if err != nil {
		t.Fatal(err)
	}
	if request.ImportPhase() != api.ImportPhaseType {
		t.Fatalf("pointer type import phase = %v", request.ImportPhase())
	}
	if _, err := api.NewRuntimeImportRequest(
		factory,
		api.ImportPhaseType,
		"../../../runtime/string.js",
		api.RuntimeStringIndex,
		"StringIndexType",
	); err == nil {
		t.Fatal("value-only string helper accepted a type-only request")
	}
}
