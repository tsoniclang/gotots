package providerboundary

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestProviderRawPointerResultPreservesCanonicalCarrier(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	input := api.DirectExpression(context.Factory().Identifier("providerPointer"))
	output, changed, err := FromProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.UnsafePointer],
		input,
	)
	if err != nil || changed || output.Value() != input.Value() || len(output.Before()) != 0 {
		t.Fatalf("canonical raw provider carrier was changed: %v", err)
	}
}

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

func TestProviderRawPointerInputPreservesCanonicalCarrier(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	input := api.DirectExpression(context.Factory().Identifier("pointer"))
	output, changed, err := ToProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.UnsafePointer],
		input,
	)
	if err != nil || changed || output.Value() != input.Value() || len(output.Before()) != 0 {
		t.Fatalf("canonical raw provider input was changed: %v", err)
	}
}

func (scalarBoundaryNames) TsonicCore(
	symbol tsoniccore.Symbol,
) (api.NameReference, error) {
	declaration, err := tsoniccore.Resolve(symbol)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(declaration.Export())
}
