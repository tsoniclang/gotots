package emit

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
)

func TestCertifiedSourceImplementationRequirementDoesNotCreateLiveness(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/certified", "certified")
	record := types.NewTypeName(token.Pos(1), sourcePackage, "Record", nil)
	consumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"Consumer",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	copyRequirement, err := api.NewNamedStructOperationRequirement(
		record,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	equalRequirement, err := api.NewNamedStructOperationRequirement(
		record,
		api.NamedStructOperationEqual,
	)
	if err != nil {
		t.Fatal(err)
	}
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	session := &programSession{
		requirements: scheduler,
		sourceImplementationContracts: map[api.ArtifactOwner]sourceImplementationContract{
			copyRequirement.Owner(): {
				acceptedRequirements: []api.DeclarationRequirement{copyRequirement},
			},
		},
	}
	if err := session.installSourceImplementationRequirements(); err != nil {
		t.Fatal(err)
	}
	if scheduler.HasPending() {
		t.Fatal("certified requirement created pending liveness")
	}
	if owner, requirements, removed, ok := scheduler.NextBatch(); ok ||
		removed || owner.Valid() || requirements != nil {
		t.Fatalf("certified requirement created a scheduler batch: %v %#v", owner, requirements)
	}
	if !scheduler.WasSelected(copyRequirement) ||
		scheduler.WasApplied(copyRequirement) {
		t.Fatal("certified requirement was not queryable without becoming applied liveness")
	}
	assertSelectedRequirements(t, scheduler, copyRequirement.Owner(), copyRequirement)

	replaceRequirements(t, scheduler, consumer, equalRequirement)
	if _, _, removed, ok := scheduler.NextBatch(); !ok || removed {
		t.Fatal("ordinary consumer requirement was not scheduled")
	}
	assertSelectedRequirements(
		t,
		scheduler,
		copyRequirement.Owner(),
		copyRequirement,
		equalRequirement,
	)
	replaceRequirements(t, scheduler, consumer)
	if !scheduler.FinalizeRemovals() {
		t.Fatal("ordinary consumer requirement removal was not scheduled")
	}
	if _, _, removed, ok := scheduler.NextBatch(); !ok || !removed {
		t.Fatal("ordinary consumer requirement removal was not applied")
	}
	assertSelectedRequirements(t, scheduler, copyRequirement.Owner(), copyRequirement)
	if !scheduler.WasSelected(copyRequirement) || scheduler.WasApplied(copyRequirement) {
		t.Fatal("liveness replacement removed or applied the certified requirement")
	}
}

func TestCertifiedSourceImplementationRequirementsRejectInvalidContracts(
	t *testing.T,
) {
	sourcePackage := types.NewPackage("example.com/certified", "certified")
	first := types.NewTypeName(token.Pos(1), sourcePackage, "First", nil)
	second := types.NewTypeName(token.Pos(2), sourcePackage, "Second", nil)
	firstRequirement, err := api.NewNamedStructOperationRequirement(
		first,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondRequirement, err := api.NewNamedStructOperationRequirement(
		second,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name         string
		contracts    map[api.ArtifactOwner]sourceImplementationContract
		wantFragment string
	}{
		{
			name: "foreign owner",
			contracts: map[api.ArtifactOwner]sourceImplementationContract{
				firstRequirement.Owner(): {
					acceptedRequirements: []api.DeclarationRequirement{secondRequirement},
				},
			},
			wantFragment: "invalid ownership",
		},
		{
			name: "duplicate",
			contracts: map[api.ArtifactOwner]sourceImplementationContract{
				firstRequirement.Owner(): {
					acceptedRequirements: []api.DeclarationRequirement{
						firstRequirement,
						firstRequirement,
					},
				},
			},
			wantFragment: "duplicated",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheduler := newDeclarationRequirementScheduler(
				emitordering.CompareArtifactOwners,
			)
			session := &programSession{
				requirements:                  scheduler,
				sourceImplementationContracts: test.contracts,
			}
			err := session.installSourceImplementationRequirements()
			if err == nil || !strings.Contains(err.Error(), test.wantFragment) {
				t.Fatalf("installation error = %v, want %q", err, test.wantFragment)
			}
			if scheduler.HasPending() || !scheduler.CertifiedEmpty() {
				t.Fatal("failed certified-requirement installation mutated scheduler state")
			}
		})
	}
}

func assertSelectedRequirements(
	t *testing.T,
	scheduler *declarationRequirementScheduler,
	owner api.ArtifactOwner,
	want ...api.DeclarationRequirement,
) {
	t.Helper()
	actual := scheduler.SelectedFor(owner)
	if len(actual) != len(want) {
		t.Fatalf("selected requirements = %#v, want %#v", actual, want)
	}
	for index := range want {
		if actual[index] != want[index] {
			t.Fatalf("selected requirements = %#v, want %#v", actual, want)
		}
	}
}
