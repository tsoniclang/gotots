package emit

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
)

func TestCommittedArtifactPlacementDropsSupersededRuntimeFeatures(t *testing.T) {
	feature, err := api.NewRuntimeFeatureRequest(api.RuntimePointerFieldPath)
	if err != nil {
		t.Fatal(err)
	}
	selected := targetplacement.New()
	if err := selected.Apply([]api.RootRequest{feature}); err != nil {
		t.Fatal(err)
	}
	builder := &targetFileBuilder{
		placement: targetplacement.New(),
		declarations: []targetDeclaration{{
			owner:     sourceArtifactOwner(artifactTestObject("Consumer", 10)),
			placement: selected,
		}},
	}
	committed, err := committedTargetFilePlacement(builder)
	if err != nil {
		t.Fatal(err)
	}
	if features := committed.RuntimeFeatures(); len(features) != 1 ||
		features[0] != api.RuntimePointerFieldPath {
		t.Fatalf("initial runtime features = %v", features)
	}
	builder.declarations[0].placement = targetplacement.New()
	committed, err = committedTargetFilePlacement(builder)
	if err != nil {
		t.Fatal(err)
	}
	if features := committed.RuntimeFeatures(); len(features) != 0 {
		t.Fatalf("superseded runtime features = %v", features)
	}
}
