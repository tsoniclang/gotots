package integer

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestUnsignedUnaryNegationIsASelectedIntegerOperation(t *testing.T) {
	carrier, ok := Describe(
		types.SizesFor("gc", "amd64"),
		types.Typ[types.Uint64],
	)
	if !ok {
		t.Fatal("uint64 carrier is absent")
	}
	for _, profile := range []api.IntegerRepresentation{
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	} {
		if !SupportsUnary(profile, carrier, token.SUB) {
			t.Fatalf("unsigned negation is unsupported under %s", profile)
		}
	}
}
