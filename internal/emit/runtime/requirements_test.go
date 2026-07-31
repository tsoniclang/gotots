package runtime

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestPackageRequirementsResolveOnlyClosedRuntimeIdentities(t *testing.T) {
	requirements, err := DecodePackageRequirements([]byte(`{
	  "schemaVersion": 1,
	  "integerRepresentations": ["number"],
	  "primitiveAliases": [
	    {"id": 4, "export": "int32"}
	  ],
	  "runtimeSymbols": [
	    {"id": 300, "export": "RuntimeSlice"}
	  ]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if !requirements.AllowsProfile(api.IntegerRepresentationNumber) ||
		requirements.AllowsProfile(api.IntegerRepresentationBigInt) {
		t.Fatal("runtime requirement profile admission is wrong")
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

func TestPackageRequirementsRejectIdentityMutations(t *testing.T) {
	for _, source := range []string{
		`{
		  "schemaVersion": 1,
		  "integerRepresentations": ["number"],
		  "primitiveAliases": [{"id": 4, "export": "uint32"}],
		  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
		}`,
		`{
		  "schemaVersion": 1,
		  "integerRepresentations": ["number"],
		  "primitiveAliases": [{"id": 4, "export": "int32"}],
		  "runtimeSymbols": [{"id": 300, "export": "GoSlice"}]
		}`,
		`{
		  "schemaVersion": 1,
		  "integerRepresentations": ["number", "number"],
		  "primitiveAliases": [{"id": 4, "export": "int32"}],
		  "runtimeSymbols": [{"id": 300, "export": "RuntimeSlice"}]
		}`,
	} {
		if _, err := DecodePackageRequirements([]byte(source)); err == nil {
			t.Fatal("mutated runtime requirement was accepted")
		}
	}
}
