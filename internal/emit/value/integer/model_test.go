package integer

import (
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestDescribePreservesNativeIntegerIdentity(t *testing.T) {
	for _, sizes := range []types.Sizes{
		&types.StdSizes{WordSize: 4, MaxAlign: 8},
		&types.StdSizes{WordSize: 8, MaxAlign: 8},
	} {
		for _, test := range []struct {
			source *types.Basic
			alias  api.PrimitiveAlias
		}{
			{types.Typ[types.Int], api.PrimitiveInt},
			{types.Typ[types.Uint], api.PrimitiveUint},
			{types.Typ[types.Uintptr], api.PrimitiveUintptr},
		} {
			carrier, ok := Describe(sizes, test.source)
			if !ok {
				t.Fatalf("describe %s at word size %d failed", test.source, sizes.Sizeof(test.source))
			}
			if carrier.Alias() != test.alias {
				t.Fatalf(
					"describe %s alias = %d, want %d",
					test.source,
					carrier.Alias(),
					test.alias,
				)
			}
		}
	}
}

func TestExactResultPolicyIsOwnedByTheIntegerProfile(t *testing.T) {
	sizes := &types.StdSizes{WordSize: 8, MaxAlign: 8}
	for _, test := range []struct {
		name          string
		profile       api.IntegerRepresentation
		source        types.Type
		exact         bool
		exactMultiply bool
	}{
		{"number int32", api.IntegerRepresentationNumber, types.Typ[types.Int32], false, false},
		{"fixed64 int32", api.IntegerRepresentationFixed64BigInt, types.Typ[types.Int32], false, false},
		{"fixed64 int64", api.IntegerRepresentationFixed64BigInt, types.Typ[types.Int64], true, false},
		{"canonical int8", api.IntegerRepresentationBigInt, types.Typ[types.Int8], true, false},
		{"canonical int32", api.IntegerRepresentationBigInt, types.Typ[types.Int32], true, true},
		{"canonical uint32", api.IntegerRepresentationBigInt, types.Typ[types.Uint32], true, true},
		{"canonical int64", api.IntegerRepresentationBigInt, types.Typ[types.Int64], true, false},
		{"canonical native", api.IntegerRepresentationBigInt, types.Typ[types.Int], true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			carrier, ok := Describe(sizes, test.source)
			if !ok {
				t.Fatalf("describe %s failed", test.source)
			}
			if actual := RequiresExactResult(test.profile, carrier); actual != test.exact {
				t.Fatalf("RequiresExactResult() = %t, want %t", actual, test.exact)
			}
			if actual := RequiresExactNumberMultiplication(test.profile, carrier); actual != test.exactMultiply {
				t.Fatalf("RequiresExactNumberMultiplication() = %t, want %t", actual, test.exactMultiply)
			}
		})
	}
}
