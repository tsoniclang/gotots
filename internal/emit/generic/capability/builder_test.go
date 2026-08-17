package capability

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestStandaloneBuilderRejectsOrdinaryConcreteOperation(t *testing.T) {
	selection, err := api.SelectGenericOperation(api.GenericOperationCopy)
	if err != nil {
		t.Fatal(err)
	}
	value := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32])
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(value),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32])),
		false,
	)
	artifact, err := api.NewCompilationGenericCapabilityArtifact(
		selection,
		signature,
		"ordinary-copy-artifact",
		"$go$copy$int32_to_int32",
		"support/generics/capabilities/copy.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	statement, requests, err := Build(api.Context{}, nil, artifact, nil)
	var invariantError *api.InvariantError
	if statement != nil || requests != nil ||
		!errors.As(err, &invariantError) ||
		invariantError.Reason !=
			"ordinary concrete generic operation reached a standalone artifact" {
		t.Fatalf(
			"standalone ordinary operation = (%T, %#v, %#v)",
			statement,
			requests,
			err,
		)
	}
}
