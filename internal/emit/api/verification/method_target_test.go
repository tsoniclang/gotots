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
		len(target.Requests()) != 1 ||
		target.Requests()[0].Kind() != api.RootRequestArtifactDependency {
		t.Fatal("method target exposed mutable request storage")
	}
}

func TestMethodTargetRejectsInvalidDomain(t *testing.T) {
	for _, testCase := range []struct {
		kind api.MethodTargetKind
		name string
	}{
		{kind: api.MethodTargetInvalid, name: "Read"},
		{kind: api.MethodTargetKind(255), name: "Read"},
		{kind: api.MethodTargetClassMember},
		{kind: api.MethodTargetEnvironmentFunction},
	} {
		if _, err := api.NewMethodTarget(testCase.kind, testCase.name); err == nil {
			t.Fatalf("NewMethodTarget(%d, %q) succeeded", testCase.kind, testCase.name)
		}
	}
}
