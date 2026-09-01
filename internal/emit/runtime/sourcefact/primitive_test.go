package sourcefact

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestPrimitiveFactOwnershipIsExact(t *testing.T) {
	tests := []struct {
		name       string
		alias      api.PrimitiveAlias
		profile    api.IntegerRepresentation
		width      api.NativeIntegerWidth
		shared     tsoniccore.Symbol
		companion  bool
		role       string
		factWidth  uint8
		factSigned string
		carrier    string
	}{
		{"fixed int32", api.PrimitiveInt32, api.IntegerRepresentationNumber, api.NativeIntegerWidth64, tsoniccore.SymbolInt32, false, "fixed", 0, "", "number"},
		{"exact int64", api.PrimitiveInt64, api.IntegerRepresentationBigInt, api.NativeIntegerWidth64, tsoniccore.SymbolInt64, false, "fixed", 0, "", "bigint"},
		{"lossy int64", api.PrimitiveInt64, api.IntegerRepresentationNumber, api.NativeIntegerWidth64, tsoniccore.SymbolInvalid, true, "fixed", 64, "signed", "number"},
		{"native int64", api.PrimitiveInt, api.IntegerRepresentationBigInt, api.NativeIntegerWidth64, tsoniccore.SymbolInt64, true, "native-int", 0, "", "bigint"},
		{"native int32", api.PrimitiveInt, api.IntegerRepresentationBigInt, api.NativeIntegerWidth32, tsoniccore.SymbolInt32, true, "native-int", 0, "", "number"},
		{"uintptr", api.PrimitiveUintptr, api.IntegerRepresentationBigInt, api.NativeIntegerWidth64, tsoniccore.SymbolUint64, true, "uintptr", 0, "", "bigint"},
		{"Go string", api.PrimitiveString, api.IntegerRepresentationBigInt, api.NativeIntegerWidth64, tsoniccore.SymbolInvalid, true, "go-string", 8, "none", "string"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			abi, err := api.NewScalarABI(test.profile, test.width)
			if err != nil {
				t.Fatal(err)
			}
			primitive, err := DescribePrimitive(test.alias, abi)
			if err != nil {
				t.Fatal(err)
			}
			if primitive.Shared != test.shared ||
				primitive.RequiresCompanion() != test.companion ||
				primitive.Role != test.role ||
				primitive.Width != test.factWidth ||
				primitive.Signed != test.factSigned ||
				primitive.Carrier != test.carrier {
				t.Fatalf("primitive = %#v", primitive)
			}
		})
	}
}

func TestSharedPrimitiveDeclarationUsesCanonicalCoreIdentity(t *testing.T) {
	abi, err := api.NewScalarABI(
		api.IntegerRepresentationBigInt,
		api.NativeIntegerWidth64,
	)
	if err != nil {
		t.Fatal(err)
	}
	primitive, err := DescribePrimitive(api.PrimitiveUint64, abi)
	if err != nil {
		t.Fatal(err)
	}
	declaration, selected, err := primitive.SharedDeclaration()
	if err != nil {
		t.Fatal(err)
	}
	if !selected ||
		declaration.Module() != "@tsonic/core/types.js" ||
		declaration.Export() != "uint64" ||
		declaration.Phase() != tsoniccore.PhaseType {
		t.Fatalf("shared declaration = %#v, selected %t", declaration, selected)
	}
}
