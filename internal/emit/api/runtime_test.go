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
		{api.RuntimeFloat32Round, 700, api.RuntimeModuleFloat, "runtime/float.ts", "goFloat32", false, nil},
		{api.RuntimeComplex64, 800, api.RuntimeModuleComplex, "runtime/complex.ts", "GoComplex64", true, []api.RuntimeSymbol{api.RuntimeFloat32Round}},
		{api.RuntimeComplex128, 801, api.RuntimeModuleComplex, "runtime/complex.ts", "GoComplex128", true, nil},
		{api.RuntimeComplexDivide, 802, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplexDivide", false, nil},
		{api.RuntimeComplex64Add, 810, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex64Add", false, []api.RuntimeSymbol{api.RuntimeComplex64}},
		{api.RuntimeComplex64Sub, 811, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex64Subtract", false, []api.RuntimeSymbol{api.RuntimeComplex64}},
		{api.RuntimeComplex64Mul, 812, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex64Multiply", false, []api.RuntimeSymbol{api.RuntimeComplex64}},
		{api.RuntimeComplex64Div, 813, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex64Divide", false, []api.RuntimeSymbol{api.RuntimeComplex64, api.RuntimeComplexDivide}},
		{api.RuntimeComplex64Neg, 814, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex64Negate", false, []api.RuntimeSymbol{api.RuntimeComplex64}},
		{api.RuntimeComplex64Equal, 815, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex64Equal", false, []api.RuntimeSymbol{api.RuntimeComplex64}},
		{api.RuntimeComplex128Add, 820, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex128Add", false, []api.RuntimeSymbol{api.RuntimeComplex128}},
		{api.RuntimeComplex128Sub, 821, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex128Subtract", false, []api.RuntimeSymbol{api.RuntimeComplex128}},
		{api.RuntimeComplex128Mul, 822, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex128Multiply", false, []api.RuntimeSymbol{api.RuntimeComplex128}},
		{api.RuntimeComplex128Div, 823, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex128Divide", false, []api.RuntimeSymbol{api.RuntimeComplex128, api.RuntimeComplexDivide}},
		{api.RuntimeComplex128Neg, 824, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex128Negate", false, []api.RuntimeSymbol{api.RuntimeComplex128}},
		{api.RuntimeComplex128Equal, 825, api.RuntimeModuleComplex, "runtime/complex.ts", "goComplex128Equal", false, []api.RuntimeSymbol{api.RuntimeComplex128}},
		{api.RuntimeNumberToBigInt, 900, api.RuntimeModuleConversion, "runtime/conversion.ts", "goNumberToBigInt", false, nil},
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
