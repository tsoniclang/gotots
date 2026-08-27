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

func TestPackageRequirementsRejectRetiredExecutionSymbols(t *testing.T) {
	for _, symbol := range []string{
		`{"id": 1105, "export": "GoScheduler"}`,
		`{"id": 1300, "export": "Awaitable"}`,
	} {
		contract, err := runtimecontract.Decode([]byte(`{
		  "schemaVersion": 3,
		  "integerRepresentations": ["number"],
		  "providerIntegerRepresentation": "number",
		  "providerScalarModule": "./internal/scalars.js",
		  "providerPointerModule": "./internal/runtime/pointer.js",
		  "nativeIntegerBits": 64,
		  "primitiveAliases": [],
		  "runtimeSymbols": [` + symbol + `]
		}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ResolvePackageRequirements(contract); err == nil {
			t.Fatalf("retired execution symbol %s was accepted", symbol)
		}
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
