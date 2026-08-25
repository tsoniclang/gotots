package runtime

import (
	"testing"

	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestPackageRequirementsResolveOnlyClosedRuntimeIdentities(t *testing.T) {
	contract, err := runtimecontract.Decode([]byte(`{
	  "schemaVersion": 3,
	  "integerRepresentations": ["number", "fixed64-bigint", "bigint"],
	  "providerIntegerRepresentation": "bigint",
	  "providerScalarModule": "./internal/scalars.js",
	  "providerPointerModule": "./internal/runtime/pointer.js",
	  "nativeIntegerBits": 64,
	  "primitiveAliases": [
	    {"id": 4, "export": "int32", "providerCarrier": "number"}
	  ],
	  "runtimeSymbols": [
	    {"id": 300, "export": "RuntimeSlice"}
	  ]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := ResolvePackageRequirements(contract)
	if err != nil {
		t.Fatal(err)
	}
	if !requirements.AllowsProfile(api.IntegerRepresentationNumber) ||
		!requirements.AllowsProfile(api.IntegerRepresentationFixed64BigInt) ||
		!requirements.AllowsProfile(api.IntegerRepresentationBigInt) {
		t.Fatal("runtime requirement profile admission is wrong")
	}
	provider := requirements.ProviderScalarABI()
	if provider.IntegerRepresentation() != api.IntegerRepresentationBigInt ||
		provider.NativeIntegerWidth() != api.NativeIntegerWidth64 {
		t.Fatal("runtime requirement provider scalar ABI is wrong")
	}
	aliases := requirements.PrimitiveAliases()
	if len(aliases) != 1 || aliases[0] != api.PrimitiveInt32 {
		t.Fatalf("runtime requirement aliases = %v", aliases)
	}
	symbols := requirements.RuntimeSymbols()
	if len(symbols) != 1 {
		t.Fatalf("runtime requirement symbols = %v", symbols)
	}
	if _, ok := symbols[api.RuntimeSlice]; !ok {
		t.Fatal("runtime requirement lost RuntimeSlice")
	}
	aliases[0] = api.PrimitiveBool
	delete(symbols, api.RuntimeSlice)
	if requirements.PrimitiveAliases()[0] != api.PrimitiveInt32 ||
		len(requirements.RuntimeSymbols()) != 1 {
		t.Fatal("runtime requirements expose mutable backing storage")
	}
}

func TestPackageRequirementsProjectExactConcurrencySurface(t *testing.T) {
	contract, err := runtimecontract.Decode([]byte(`{
	  "schemaVersion": 3,
	  "integerRepresentations": ["number"],
	  "providerIntegerRepresentation": "number",
	  "providerScalarModule": "./internal/scalars.js",
	  "providerPointerModule": "./internal/runtime/pointer.js",
	  "nativeIntegerBits": 64,
	  "primitiveAliases": [],
	  "runtimeSymbols": [
	    {"id": 300, "export": "RuntimeSlice"},
	    {"id": 1105, "export": "GoScheduler"},
	    {"id": 1300, "export": "Awaitable"}
	  ]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := ResolvePackageRequirements(contract)
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := requirements.RuntimeSymbolsFor(
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 1 {
		t.Fatalf("disabled runtime symbols = %v, want only RuntimeSlice", disabled)
	}
	if _, ok := disabled[api.RuntimeSlice]; !ok {
		t.Fatal("disabled runtime projection lost RuntimeSlice")
	}
	cooperative, err := requirements.RuntimeSymbolsFor(
		api.ConcurrencySemanticsCooperative,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(cooperative) != 3 {
		t.Fatalf("cooperative runtime symbols = %v, want all three", cooperative)
	}
	if len(requirements.RuntimeSymbols()) != 3 {
		t.Fatal("runtime projection mutated the certified source requirements")
	}
	if _, err := requirements.RuntimeSymbolsFor(
		api.ConcurrencySemanticsInvalid,
	); err == nil {
		t.Fatal("invalid concurrency projected a runtime surface")
	}
}

func TestPackageRequirementsRejectIdentityMutations(t *testing.T) {
	for _, source := range []string{
		`{
		  "schemaVersion": 3,
		  "integerRepresentations": ["number", "bigint"],
		  "providerIntegerRepresentation": "bigint",
		  "providerScalarModule": "./internal/scalars.js",
		  "providerPointerModule": "./internal/runtime/pointer.js",
		  "nativeIntegerBits": 64,
		  "primitiveAliases": [{"id": 4, "export": "uint32", "providerCarrier": "number"}],
		  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
		}`,
		`{
		  "schemaVersion": 3,
		  "integerRepresentations": ["number", "bigint"],
		  "providerIntegerRepresentation": "bigint",
		  "providerScalarModule": "./internal/scalars.js",
		  "providerPointerModule": "./internal/runtime/pointer.js",
		  "nativeIntegerBits": 64,
		  "primitiveAliases": [{"id": 4, "export": "int32", "providerCarrier": "number"}],
		  "runtimeSymbols": [{"id": 300, "export": "GoSlice"}]
		}`,
		`{
		  "schemaVersion": 3,
		  "integerRepresentations": ["number", "number"],
		  "providerIntegerRepresentation": "number",
		  "providerScalarModule": "./internal/scalars.js",
		  "providerPointerModule": "./internal/runtime/pointer.js",
		  "nativeIntegerBits": 64,
		  "primitiveAliases": [{"id": 4, "export": "int32", "providerCarrier": "number"}],
		  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
		}`,
	} {
		contract, err := runtimecontract.Decode([]byte(source))
		if err == nil {
			_, err = ResolvePackageRequirements(contract)
		}
		if err == nil {
			t.Fatal("mutated runtime requirement was accepted")
		}
	}
}
