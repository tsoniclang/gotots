package tsoniccore

import "testing"

func TestPointerContractIsPinned(t *testing.T) {
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
