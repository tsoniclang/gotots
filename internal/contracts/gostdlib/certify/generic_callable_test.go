package certify

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestGenericKernelValueArityJoinsCapabilitiesAndGoParameters(t *testing.T) {
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "Element", nil),
		types.Universe.Lookup("any").Type(),
	)
	signature := types.NewSignatureType(
		nil,
		nil,
		[]*types.TypeParam{parameter},
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "left", parameter),
			types.NewVar(token.NoPos, nil, "right", parameter),
		),
		types.NewTuple(types.NewVar(token.NoPos, nil, "result", parameter)),
		false,
	)
	projection := []gostdlib.GenericTypeArgumentDocument{{
		TypeParameter: 0,
		Facet:         gostdlib.GenericTypeArgumentLogical,
	}}
	operations := []gostdlib.GenericOperationDocument{{
		Kind: gostdlib.GenericOperationEqual,
	}}
	if err := verifyGenericKernelShape(
		"example.com/source|Equal",
		signature,
		projection,
		operations,
		1,
		3,
	); err != nil {
		t.Fatal(err)
	}
	err := verifyGenericKernelShape(
		"example.com/source|Equal",
		signature,
		projection,
		operations,
		1,
		4,
	)
	if err == nil || !strings.Contains(
		err.Error(),
		"kernel has 4 value parameters, capability and source contract requires 3",
	) {
		t.Fatalf("hidden kernel-parameter mutation error = %v", err)
	}
}
