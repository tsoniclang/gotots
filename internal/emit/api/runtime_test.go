package api_test

import (
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
	}{
		{api.RuntimeStringIndex, 1, api.RuntimeModuleString, "runtime/string.ts", "goStringIndex", false},
		{api.RuntimeStringSlice, 2, api.RuntimeModuleString, "runtime/string.ts", "goStringSlice", false},
		{api.RuntimePointer, 100, api.RuntimeModulePointer, "runtime/pointer.ts", "GoPointer", true},
		{api.RuntimeArray, 200, api.RuntimeModuleArray, "runtime/array.ts", "GoArray", true},
		{api.RuntimeSlice, 300, api.RuntimeModuleSlice, "runtime/slice.ts", "GoSlice", true},
		{api.RuntimeMap, 400, api.RuntimeModuleMap, "runtime/map.ts", "GoMap", true},
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
	if request.Kind() != api.PlacementImport ||
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
