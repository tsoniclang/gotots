package api_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestMethodTargetIsClosedAndImmutable(t *testing.T) {
	provider := types.NewFunc(
		token.Pos(1),
		types.NewPackage("example.com/source", "source"),
		"Read",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	request, err := api.NewArtifactDependencyRequest(
		provider,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	requests := []api.RootRequest{request}
	target, err := api.NewMethodTarget(
		api.MethodTargetClassMember,
		"Read",
		api.MethodReceiverABIContractDirect,
		requests...,
	)
	if err != nil {
		t.Fatal(err)
	}
	requests[0] = api.RootRequest{}
	returned := target.Requests()
	returned[0] = api.RootRequest{}
	if target.Kind() != api.MethodTargetClassMember ||
		target.Name() != "Read" ||
		target.ReceiverABI() != api.MethodReceiverABIContractDirect ||
		len(target.Requests()) != 1 ||
		target.Requests()[0].Kind() != api.RootRequestArtifactDependency {
		t.Fatal("method target exposed mutable request storage")
	}
}

func TestMethodTargetRejectsInvalidDomain(t *testing.T) {
	for _, testCase := range []struct {
		kind api.MethodTargetKind
		name string
		abi  api.MethodReceiverABI
	}{
		{kind: api.MethodTargetInvalid, name: "Read", abi: api.MethodReceiverABISourceRepresentation},
		{kind: api.MethodTargetKind(255), name: "Read", abi: api.MethodReceiverABISourceRepresentation},
		{kind: api.MethodTargetClassMember, abi: api.MethodReceiverABISourceRepresentation},
		{kind: api.MethodTargetEnvironmentFunction, abi: api.MethodReceiverABISourceRepresentation},
		{kind: api.MethodTargetClassMember, name: "Read"},
		{kind: api.MethodTargetClassMember, name: "Read", abi: api.MethodReceiverABI(255)},
	} {
		if _, err := api.NewMethodTarget(testCase.kind, testCase.name, testCase.abi); err == nil {
			t.Fatalf("NewMethodTarget(%d, %q, %d) succeeded", testCase.kind, testCase.name, testCase.abi)
		}
	}
}
