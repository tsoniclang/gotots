package providerboundary

import (
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestProviderRawPointerResultBindsOpaqueIdentity(t *testing.T) {
	context := scalarBoundaryContext(
		t,
		"amd64",
		api.IntegerRepresentationNumber,
		api.IntegerRepresentationBigInt,
	)
	target, changed, err := FromProviderValue(
		context,
		nil,
		nil,
		"",
		types.Typ[types.UnsafePointer],
		api.DirectExpression(context.Factory().Identifier("providerPointer")),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("provider raw-pointer result bypassed canonical identity binding")
	}
	conditional, ok := target.Value().(tsgo.ConditionalExpression)
	if !ok {
		t.Fatalf("provider raw-pointer bridge = %T, want nil-preserving conditional", target.Value())
	}
	call, ok := conditional.WhenFalse().(tsgo.CallExpression)
	if !ok {
		t.Fatalf("provider raw-pointer non-nil branch = %T, want marker call", conditional.WhenFalse())
	}
	callee, ok := call.Expression().(tsgo.Identifier)
	if !ok || callee.Text() != "bindRawPointer" {
		t.Fatalf("provider raw-pointer callee = %T %v", call.Expression(), call.Expression())
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
