package emit

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
)

func TestRequirementRemovalWaitsForQuiescentConsumerDiscovery(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/liveness", "liveness")
	provider := types.NewTypeName(token.Pos(1), sourcePackage, "Record", nil)
	firstConsumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"First",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	laterConsumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(3),
		sourcePackage,
		"Later",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	copyRequirement, err := api.NewNamedStructOperationRequirement(
		provider,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	equalRequirement, err := api.NewNamedStructOperationRequirement(
		provider,
		api.NamedStructOperationEqual,
	)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	scheduler.replace(
		firstConsumer,
		[]api.DeclarationRequirement{copyRequirement},
	)
	_, _, _, _ = scheduler.nextBatch()

	scheduler.replace(
		firstConsumer,
		[]api.DeclarationRequirement{equalRequirement},
	)
	owner, requirements, removed, ok := scheduler.nextBatch()
	if !ok ||
		removed ||
		owner != copyRequirement.Owner() ||
		len(requirements) != 2 {
		t.Fatalf(
			"replacement addition = %v %#v removed=%t ok=%t",
			owner,
			requirements,
			removed,
			ok,
		)
	}
	scheduler.replace(
		laterConsumer,
		[]api.DeclarationRequirement{copyRequirement},
	)
	if scheduler.finalizeRemovals() {
		t.Fatal("later consumer did not cancel the pending removal")
	}
	if scheduler.hasPending() {
		t.Fatal("canceled removal left pending scheduler work")
	}
}
