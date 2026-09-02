package representationcontract

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestBigIntProfileSelectsCarrierByGoWidth(t *testing.T) {
	width32, err := NewScalarABI(
		IntegerRepresentationBigInt,
		NativeIntegerWidth32,
	)
	if err != nil {
		t.Fatal(err)
	}
	width64, err := NewScalarABI(
		IntegerRepresentationBigInt,
		NativeIntegerWidth64,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		alias   PrimitiveAlias
		width32 IntegerCarrier
		width64 IntegerCarrier
	}{
		{PrimitiveInt8, IntegerCarrierNumber, IntegerCarrierNumber},
		{PrimitiveInt16, IntegerCarrierNumber, IntegerCarrierNumber},
		{PrimitiveInt32, IntegerCarrierNumber, IntegerCarrierNumber},
		{PrimitiveUint8, IntegerCarrierNumber, IntegerCarrierNumber},
		{PrimitiveUint16, IntegerCarrierNumber, IntegerCarrierNumber},
		{PrimitiveUint32, IntegerCarrierNumber, IntegerCarrierNumber},
		{PrimitiveInt64, IntegerCarrierBigInt, IntegerCarrierBigInt},
		{PrimitiveUint64, IntegerCarrierBigInt, IntegerCarrierBigInt},
		{PrimitiveInt, IntegerCarrierNumber, IntegerCarrierBigInt},
		{PrimitiveUint, IntegerCarrierNumber, IntegerCarrierBigInt},
		{PrimitiveUintptr, IntegerCarrierNumber, IntegerCarrierBigInt},
	} {
		got32, err := width32.Carrier(test.alias)
		if err != nil {
			t.Fatalf("32-bit carrier for alias %d: %v", test.alias, err)
		}
		got64, err := width64.Carrier(test.alias)
		if err != nil {
			t.Fatalf("64-bit carrier for alias %d: %v", test.alias, err)
		}
		if got32 != test.width32 || got64 != test.width64 {
			t.Fatalf(
				"carrier for alias %d = %d/%d, want %d/%d",
				test.alias,
				got32,
				got64,
				test.width32,
				test.width64,
			)
		}
	}
}

func TestFixed64BigIntProfileLeavesNativeIntegersAsNumbers(t *testing.T) {
	abi, err := NewScalarABI(
		IntegerRepresentationFixed64BigInt,
		NativeIntegerWidth64,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		alias PrimitiveAlias
		want  IntegerCarrier
	}{
		{PrimitiveInt64, IntegerCarrierBigInt},
		{PrimitiveUint64, IntegerCarrierBigInt},
		{PrimitiveInt, IntegerCarrierNumber},
		{PrimitiveUint, IntegerCarrierNumber},
		{PrimitiveUintptr, IntegerCarrierNumber},
	} {
		got, err := abi.Carrier(test.alias)
		if err != nil {
			t.Fatalf("carrier for alias %d: %v", test.alias, err)
		}
		if got != test.want {
			t.Fatalf("carrier for alias %d = %d, want %d", test.alias, got, test.want)
		}
	}
}

func TestScalarABISelectsExactIntegerResultPolicy(t *testing.T) {
	for _, test := range []struct {
		name    string
		profile IntegerRepresentation
		alias   PrimitiveAlias
		want    bool
	}{
		{"number int32", IntegerRepresentationNumber, PrimitiveInt32, false},
		{"fixed64 int32", IntegerRepresentationFixed64BigInt, PrimitiveInt32, false},
		{"fixed64 int64", IntegerRepresentationFixed64BigInt, PrimitiveInt64, true},
		{"canonical int8", IntegerRepresentationBigInt, PrimitiveInt8, true},
		{"canonical int64", IntegerRepresentationBigInt, PrimitiveInt64, true},
		{"non-integer", IntegerRepresentationBigInt, PrimitiveBool, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			abi, err := NewScalarABI(test.profile, NativeIntegerWidth64)
			if err != nil {
				t.Fatal(err)
			}
			if actual := abi.RequiresExactIntegerResult(test.alias); actual != test.want {
				t.Fatalf("RequiresExactIntegerResult() = %t, want %t", actual, test.want)
			}
		})
	}
}

func TestPrimitiveAliasIDsAndNamesPreserveSourceIdentity(t *testing.T) {
	for _, test := range []struct {
		alias PrimitiveAlias
		id    PrimitiveAlias
		name  string
	}{
		{PrimitiveInt8, 2, "int8"},
		{PrimitiveInt64, 5, "int64"},
		{PrimitiveUint64, 9, "uint64"},
		{PrimitiveFloat64, 12, "float64"},
		{PrimitiveInt, 13, "int"},
		{PrimitiveUint, 14, "uint"},
		{PrimitiveUintptr, 15, "uintptr"},
	} {
		name, err := PrimitiveAliasName(test.alias)
		if err != nil {
			t.Fatal(err)
		}
		if test.alias != test.id || name != test.name {
			t.Fatalf("alias %d name = %q, want %d/%q", test.alias, name, test.id, test.name)
		}
	}
}

func TestIntegerLiteralUsesSelectedCarrier(t *testing.T) {
	factory := tsgo.NewFactory()
	abi, err := NewScalarABI(
		IntegerRepresentationBigInt,
		NativeIntegerWidth64,
	)
	if err != nil {
		t.Fatal(err)
	}
	narrow, err := IntegerLiteral(
		factory,
		abi,
		PrimitiveUint8,
		"1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if narrow.Kind() != tsgo.SyntaxKindNumericLiteral {
		t.Fatalf("uint8 literal kind = %d, want numeric", narrow.Kind())
	}
	wide, err := IntegerLiteral(
		factory,
		abi,
		PrimitiveUint64,
		"1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if wide.Kind() != tsgo.SyntaxKindBigIntLiteral {
		t.Fatalf("uint64 literal kind = %d, want BigInt", wide.Kind())
	}
}
