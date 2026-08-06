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
