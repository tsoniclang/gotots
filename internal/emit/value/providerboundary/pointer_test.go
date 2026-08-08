package providerboundary

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestProviderScalarPointerInputFailsWithoutExactInverseTransport(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	_, _, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		types.NewPointer(types.Typ[types.Int]),
		api.DirectExpression(context.Factory().Identifier("pointer")),
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"exact external-location transport contract",
	) {
		t.Fatalf("scalar provider-pointer input error = %v", err)
	}
}
