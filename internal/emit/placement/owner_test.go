package placement

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRuntimeFeatureSelectsSupportWithoutEmittingAnImport(t *testing.T) {
	request, err := api.NewRuntimeFeatureRequest(api.RuntimePointerFieldPath)
	if err != nil {
		t.Fatal(err)
	}
	owner := New()
	if err := owner.Apply([]api.RootRequest{request, request}); err != nil {
		t.Fatal(err)
	}
	features := owner.RuntimeFeatures()
	if len(features) != 1 || features[0] != api.RuntimePointerFieldPath {
		t.Fatalf("runtime features = %v", features)
	}
	if statements := owner.Statements(tsgo.NewFactory()); len(statements) != 0 {
		t.Fatalf("runtime feature emitted imports: %#v", statements)
	}
	if err := owner.RequireTypeOnly(); err != nil {
		t.Fatalf("runtime feature violates type-only placement: %v", err)
	}
}
