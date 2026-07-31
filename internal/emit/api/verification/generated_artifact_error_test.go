package api_test

import (
	"errors"
	"go/token"
	"go/types"
	"strings"
	"testing"

	. "github.com/tsoniclang/gotots/internal/emit/api"
)

func TestGeneratedCapabilityErrorNamesItsSemanticOperation(t *testing.T) {
	signature := types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32]),
		),
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32]),
		),
		false,
	)
	selection, err := SelectGenericOperation(GenericOperationCopy)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewCompilationGenericCapabilityArtifact(
		selection,
		signature,
		"copy-int32",
		"$goCapability_copy_int32",
		"support/generics/capabilities/copy-int32.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	message := WrapGeneratedArtifactError(
		artifact,
		errors.New("failed"),
	).Error()
	if !strings.Contains(message, "operation copy") {
		t.Fatalf("generated capability diagnostic = %q", message)
	}
}
