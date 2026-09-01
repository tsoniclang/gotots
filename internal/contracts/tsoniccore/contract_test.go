package tsoniccore

import "testing"

func TestContractIsPinned(t *testing.T) {
	tests := []struct {
		symbol Symbol
		module string
		export string
		phase  Phase
	}{
		{SymbolPointer, "@tsonic/core/types.js", "Pointer", PhaseType},
		{SymbolAddressOf, "@tsonic/core/lang.js", "addressOf", PhaseValue},
		{SymbolAllocatePointer, "@tsonic/core/lang.js", "allocatePointer", PhaseValue},
		{SymbolLoadPointer, "@tsonic/core/lang.js", "loadPointer", PhaseValue},
		{SymbolStorePointer, "@tsonic/core/lang.js", "storePointer", PhaseValue},
		{SymbolEqualPointer, "@tsonic/core/lang.js", "equalPointer", PhaseValue},
		{SymbolHashPointer, "@tsonic/core/lang.js", "hashPointer", PhaseValue},
		{SymbolProjectPointer, "@tsonic/core/lang.js", "projectPointer", PhaseValue},
		{SymbolBindPointer, "@tsonic/core/lang.js", "bindPointer", PhaseValue},
		{SymbolRawPointer, "@tsonic/core/types.js", "RawPointer", PhaseType},
		{SymbolBindRawPointer, "@tsonic/core/lang.js", "bindRawPointer", PhaseValue},
		{SymbolEqualRawPointer, "@tsonic/core/lang.js", "equalRawPointer", PhaseValue},
		{SymbolHashRawPointer, "@tsonic/core/lang.js", "hashRawPointer", PhaseValue},
		{SymbolBool, "@tsonic/core/types.js", "bool", PhaseType},
		{SymbolInt8, "@tsonic/core/types.js", "int8", PhaseType},
		{SymbolUint8, "@tsonic/core/types.js", "uint8", PhaseType},
		{SymbolInt16, "@tsonic/core/types.js", "int16", PhaseType},
		{SymbolUint16, "@tsonic/core/types.js", "uint16", PhaseType},
		{SymbolInt32, "@tsonic/core/types.js", "int32", PhaseType},
		{SymbolUint32, "@tsonic/core/types.js", "uint32", PhaseType},
		{SymbolInt64, "@tsonic/core/types.js", "int64", PhaseType},
		{SymbolUint64, "@tsonic/core/types.js", "uint64", PhaseType},
		{SymbolFloat32, "@tsonic/core/types.js", "float32", PhaseType},
		{SymbolFloat64, "@tsonic/core/types.js", "float64", PhaseType},
	}
	for _, test := range tests {
		declaration, err := Resolve(test.symbol)
		if err != nil {
			t.Fatal(err)
		}
		if declaration.Module() != test.module ||
			declaration.Export() != test.export ||
			declaration.Phase() != test.phase {
			t.Fatalf(
				"symbol %d = %q/%q/%d, want %q/%q/%d",
				test.symbol,
				declaration.Module(),
				declaration.Export(),
				declaration.Phase(),
				test.module,
				test.export,
				test.phase,
			)
		}
	}
	if _, err := Resolve(SymbolInvalid); err == nil {
		t.Fatal("invalid symbol resolved")
	}
}
