package providerboundary

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestProviderRawPointerResultRequiresAddressContract(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	_, _, err := FromProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.UnsafePointer],
		api.DirectExpression(context.Factory().Identifier("providerPointer")),
	)
	if err == nil || !strings.Contains(err.Error(), "exact address-bearing provider contract") {
		t.Fatalf("raw provider object accepted without address contract: %v", err)
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

func TestProviderRawPointerInputFailsWithoutIdentityExtraction(t *testing.T) {
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
		types.Typ[types.UnsafePointer],
		api.DirectExpression(context.Factory().Identifier("pointer")),
	)
	if err == nil || !strings.Contains(err.Error(), "raw-pointer identity extraction") {
		t.Fatalf("provider raw-pointer input error = %v", err)
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
