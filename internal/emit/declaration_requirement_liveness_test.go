package emit

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
)

func replaceRequirements(
	t *testing.T,
	scheduler *declarationRequirementScheduler,
	consumer api.ArtifactOwner,
	requirements ...api.DeclarationRequirement,
) {
	t.Helper()
	requests := make([]api.RootRequest, 0, len(requirements))
	for _, requirement := range requirements {
		request, err := api.NewDeclarationRequirementRequest(requirement)
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, request)
	}
	if err := scheduler.Replace(consumer, requests); err != nil {
		t.Fatal(err)
	}
}

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
	replaceRequirements(t, scheduler, firstConsumer, copyRequirement)
	_, _, _, _ = scheduler.NextBatch()

	replaceRequirements(t, scheduler, firstConsumer, equalRequirement)
	owner, requirements, removed, ok := scheduler.NextBatch()
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
	replaceRequirements(t, scheduler, laterConsumer, copyRequirement)
	if scheduler.FinalizeRemovals() {
		t.Fatal("later consumer did not cancel the pending removal")
	}
	if scheduler.HasPending() {
		t.Fatal("canceled removal left pending scheduler work")
	}
}
