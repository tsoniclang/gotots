package integer

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestDescribePreservesEveryPredeclaredIntegerWidthAndSign(t *testing.T) {
	testCases := []struct {
		name   string
		kind   types.BasicKind
		arch   string
		alias  api.PrimitiveAlias
		width  uint8
		signed bool
	}{
		{"int8", types.Int8, "amd64", api.PrimitiveInt8, 8, true},
		{"int16", types.Int16, "amd64", api.PrimitiveInt16, 16, true},
		{"int32", types.Int32, "amd64", api.PrimitiveInt32, 32, true},
		{"int64", types.Int64, "amd64", api.PrimitiveInt64, 64, true},
		{"uint8", types.Uint8, "amd64", api.PrimitiveUint8, 8, false},
		{"uint16", types.Uint16, "amd64", api.PrimitiveUint16, 16, false},
		{"uint32", types.Uint32, "amd64", api.PrimitiveUint32, 32, false},
		{"uint64", types.Uint64, "amd64", api.PrimitiveUint64, 64, false},
		{"int-386", types.Int, "386", api.PrimitiveInt32, 32, true},
		{"int-amd64", types.Int, "amd64", api.PrimitiveInt64, 64, true},
		{"uint-386", types.Uint, "386", api.PrimitiveUint32, 32, false},
		{"uint-amd64", types.Uint, "amd64", api.PrimitiveUint64, 64, false},
		{"uintptr-386", types.Uintptr, "386", api.PrimitiveUint32, 32, false},
		{"uintptr-amd64", types.Uintptr, "amd64", api.PrimitiveUint64, 64, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, ok := Describe(
				types.SizesFor("gc", testCase.arch),
				types.Typ[testCase.kind],
			)
			if !ok ||
				got.Alias() != testCase.alias ||
				got.Width() != testCase.width ||
				got.Signed() != testCase.signed {
				t.Fatalf("carrier = %#v, %v", got, ok)
			}
		})
	}
}

func TestFormatConstantEnforcesCarrierBoundsAndSelectedSyntax(t *testing.T) {
	int64Carrier, _ := Describe(
		types.SizesFor("gc", "amd64"),
		types.Typ[types.Int64],
	)
	uint64Carrier, _ := Describe(
		types.SizesFor("gc", "amd64"),
		types.Typ[types.Uint64],
	)
	testCases := []struct {
		name           string
		representation api.IntegerRepresentation
		carrier        Carrier
		value          string
		wantMagnitude  string
		wantNegative   bool
		want           bool
	}{
		{"number-safe", api.IntegerRepresentationNumber, int64Carrier, "9007199254740991", "9007199254740991", false, true},
		{"number-wide", api.IntegerRepresentationNumber, int64Carrier, "9007199254740992", "9007199254740992", false, true},
		{"bigint-max-signed", api.IntegerRepresentationBigInt, int64Carrier, "9223372036854775807", "9223372036854775807", false, true},
		{"bigint-min-signed", api.IntegerRepresentationBigInt, int64Carrier, "-9223372036854775808", "9223372036854775808", true, true},
		{"bigint-max-unsigned", api.IntegerRepresentationBigInt, uint64Carrier, "18446744073709551615", "18446744073709551615", false, true},
		{"unsigned-negative", api.IntegerRepresentationBigInt, uint64Carrier, "-1", "", false, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			value := constant.MakeFromLiteral(testCase.value, token.INT, 0)
			magnitude, negative, ok := FormatConstant(
				testCase.representation,
				testCase.carrier,
				value,
			)
			if magnitude != testCase.wantMagnitude ||
				negative != testCase.wantNegative ||
				ok != testCase.want {
				t.Fatalf(
					"format = %q, %v, %v; want %q, %v, %v",
					magnitude,
					negative,
					ok,
					testCase.wantMagnitude,
					testCase.wantNegative,
					testCase.want,
				)
			}
		})
	}
}

func TestEveryCarrierBoundaryIsCheckedAgainstItsSelectedProfile(t *testing.T) {
	testCases := []struct {
		kind    types.BasicKind
		minimum string
		maximum string
	}{
		{types.Int8, "-128", "127"},
		{types.Int16, "-32768", "32767"},
		{types.Int32, "-2147483648", "2147483647"},
		{types.Int64, "-9223372036854775808", "9223372036854775807"},
		{types.Uint8, "0", "255"},
		{types.Uint16, "0", "65535"},
		{types.Uint32, "0", "4294967295"},
		{types.Uint64, "0", "18446744073709551615"},
	}
	sizes := types.SizesFor("gc", "amd64")
	for _, testCase := range testCases {
		carrier, ok := Describe(sizes, types.Typ[testCase.kind])
		if !ok {
			t.Fatalf("carrier %v is absent", testCase.kind)
		}
		for _, boundary := range []string{testCase.minimum, testCase.maximum} {
			value := constant.MakeFromLiteral(boundary, token.INT, 0)
			if _, _, exact := FormatConstant(
				api.IntegerRepresentationBigInt,
				carrier,
				value,
			); !exact {
				t.Fatalf("BigInt %v boundary %s was rejected", testCase.kind, boundary)
			}
			if _, _, admitted := FormatConstant(
				api.IntegerRepresentationNumber,
				carrier,
				value,
			); !admitted {
				t.Fatalf(
					"number %v boundary %s was rejected",
					testCase.kind,
					boundary,
				)
			}
		}
	}
}

func TestIntegerCapabilityMatrixMatchesSelectedProfileBoundaries(t *testing.T) {
	sizes := types.SizesFor("gc", "amd64")
	int32Carrier, _ := Describe(sizes, types.Typ[types.Int32])
	int64Carrier, _ := Describe(sizes, types.Typ[types.Int64])
	uint32Carrier, _ := Describe(sizes, types.Typ[types.Uint32])
	two := constant.MakeInt64(2)
	if !SupportsArithmetic(api.IntegerRepresentationNumber, token.QUO) ||
		!SupportsArithmetic(api.IntegerRepresentationNumber, token.REM) {
		t.Fatal("number integer division or remainder was rejected")
	}
	if !SupportsArithmetic(api.IntegerRepresentationBigInt, token.QUO) {
		t.Fatal("BigInt division was rejected")
	}
	if !SupportsBitwise(api.IntegerRepresentationNumber, int64Carrier, token.AND) {
		t.Fatal("number int64 bitwise operation was rejected")
	}
	if !SupportsBitwise(api.IntegerRepresentationNumber, int32Carrier, token.AND) {
		t.Fatal("number int32 bitwise operation was rejected")
	}
	if SupportsBitwise(api.IntegerRepresentationInvalid, int64Carrier, token.AND) {
		t.Fatal("invalid integer representation admitted bitwise operation")
	}
	if SupportsBitwise(api.IntegerRepresentationNumber, int64Carrier, token.ADD) {
		t.Fatal("non-bitwise operator was admitted as bitwise")
	}
	if !RequiresUint32Normalization(api.IntegerRepresentationNumber, uint32Carrier) {
		t.Fatal("number uint32 lost its required unsigned normalization")
	}
	if !SupportsShift(api.IntegerRepresentationBigInt, int64Carrier, token.SHL, two) {
		t.Fatal("constant BigInt shift was rejected")
	}
	if SupportsShift(api.IntegerRepresentationBigInt, int64Carrier, token.SHL, nil) {
		t.Fatal("constant-count capability accepted absent evidence")
	}
	if !SupportsVariableShift(
		api.IntegerRepresentationBigInt,
		int64Carrier,
		token.SHL,
	) {
		t.Fatal("exact BigInt variable shift was rejected")
	}
	if !SupportsVariableShift(
		api.IntegerRepresentationNumber,
		int64Carrier,
		token.SHL,
	) {
		t.Fatal("number int64 variable shift was rejected")
	}
	if !SupportsVariableShift(
		api.IntegerRepresentationNumber,
		int32Carrier,
		token.SHR,
	) {
		t.Fatal("exact number int32 variable shift was rejected")
	}
	if SupportsVariableShift(
		api.IntegerRepresentationNumber,
		int64Carrier,
		token.AND,
	) {
		t.Fatal("non-shift operator was admitted as a variable shift")
	}
	if !SupportsUnary(
		api.IntegerRepresentationNumber,
		int64Carrier,
		token.XOR,
	) {
		t.Fatal("number int64 complement was rejected")
	}
	if SupportsUnary(api.IntegerRepresentationBigInt, uint32Carrier, token.SUB) {
		t.Fatal("unsigned negation was admitted without fixed-width overflow")
	}
	if mask, ok := UnsignedMask(uint32Carrier); !ok || mask != "4294967295" {
		t.Fatalf("uint32 mask = %q, %v", mask, ok)
	}
}

func TestCanConvertDirectlyOnlyWhenEverySourceValueFitsTarget(t *testing.T) {
	sizes := types.SizesFor("gc", "amd64")
	carrier := func(kind types.BasicKind) Carrier {
		result, ok := Describe(sizes, types.Typ[kind])
		if !ok {
			t.Fatalf("carrier %v is absent", kind)
		}
		return result
	}
	for _, testCase := range []struct {
		source types.BasicKind
		target types.BasicKind
		want   bool
	}{
		{types.Int8, types.Int64, true},
		{types.Uint32, types.Int64, true},
		{types.Uint8, types.Uint64, true},
		{types.Int64, types.Int8, false},
		{types.Int8, types.Uint64, false},
		{types.Uint64, types.Int64, false},
	} {
		if got := CanConvertDirectly(
			carrier(testCase.source),
			carrier(testCase.target),
		); got != testCase.want {
			t.Fatalf("convert %v -> %v = %v, want %v", testCase.source, testCase.target, got, testCase.want)
		}
	}
}
