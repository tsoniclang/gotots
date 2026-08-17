package emit

import (
	"fmt"
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
	if err := scheduler.replace(consumer, requests); err != nil {
		t.Fatal(err)
	}
}

func TestRequirementSchedulerSharesPersistentRequestClosure(t *testing.T) {
	const (
		requirementCount = 32
		consumerCount    = 1_000
	)
	sourcePackage := types.NewPackage("example.com/shared", "shared")
	requests := make([]api.RootRequest, 0, requirementCount)
	for index := 0; index < requirementCount; index++ {
		typeName := types.NewTypeName(
			token.Pos(index+1),
			sourcePackage,
			fmt.Sprintf("Value%d", index),
			nil,
		)
		requirement, err := api.NewNamedStructOperationRequirement(
			typeName,
			api.NamedStructOperationCopy,
		)
		if err != nil {
			t.Fatal(err)
		}
		request, err := api.NewDeclarationRequirementRequest(requirement)
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, request)
	}
	shared := api.CombineRequests(requests)
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	consumers := make([]api.ArtifactOwner, 0, consumerCount)
	for index := 0; index < consumerCount; index++ {
		consumer := api.MustSourceArtifactOwner(types.NewFunc(
			token.Pos(requirementCount+index+1),
			sourcePackage,
			fmt.Sprintf("Consumer%d", index),
			types.NewSignatureType(nil, nil, nil, nil, nil, false),
		))
		consumers = append(consumers, consumer)
		if err := scheduler.replace(consumer, shared); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(scheduler.requestRefs); got != requirementCount+1 {
		t.Fatalf(
			"retained request nodes = %d, want %d shared nodes",
			got,
			requirementCount+1,
		)
	}
	if got := len(scheduler.requirementRefs); got != requirementCount {
		t.Fatalf(
			"retained requirements = %d, want %d",
			got,
			requirementCount,
		)
	}
	for _, consumer := range consumers[:len(consumers)-1] {
		if err := scheduler.replace(consumer, nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(scheduler.orphaned) != 0 {
		t.Fatal("shared closure became orphaned while one consumer remained")
	}
	if err := scheduler.replace(consumers[len(consumers)-1], nil); err != nil {
		t.Fatal(err)
	}
	if len(scheduler.orphaned) != requirementCount {
		t.Fatalf(
			"orphaned requirements = %d, want %d",
			len(scheduler.orphaned),
			requirementCount,
		)
	}
}

func TestRequirementSchedulerReplacesAroundSharedSubgraph(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/replacement", "replacement")
	newRequirement := func(name string, position token.Pos) api.DeclarationRequirement {
		t.Helper()
		typeName := types.NewTypeName(position, sourcePackage, name, nil)
		requirement, err := api.NewNamedStructOperationRequirement(
			typeName,
			api.NamedStructOperationCopy,
		)
		if err != nil {
			t.Fatal(err)
		}
		return requirement
	}
	newRequest := func(requirement api.DeclarationRequirement) api.RootRequest {
		t.Helper()
		request, err := api.NewDeclarationRequirementRequest(requirement)
		if err != nil {
			t.Fatal(err)
		}
		return request
	}
	sharedRequirement := newRequirement("Shared", token.Pos(1))
	removedRequirement := newRequirement("Removed", token.Pos(2))
	addedRequirement := newRequirement("Added", token.Pos(3))
	shared := api.CombineRequests([]api.RootRequest{
		newRequest(sharedRequirement),
	})
	before := api.CombineRequests(
		shared,
		[]api.RootRequest{newRequest(removedRequirement)},
	)
	after := api.CombineRequests(
		shared,
		[]api.RootRequest{newRequest(addedRequirement)},
	)
	consumer := api.MustSourceArtifactOwner(types.NewFunc(
		token.Pos(4),
		sourcePackage,
		"Consumer",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	))
	scheduler := newDeclarationRequirementScheduler(
		emitordering.CompareArtifactOwners,
	)
	if err := scheduler.replace(consumer, before); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.replace(consumer, after); err != nil {
		t.Fatal(err)
	}
	if _, orphaned := scheduler.orphaned[sharedRequirement]; orphaned {
		t.Fatal("shared requirement became transiently orphaned during replacement")
	}
	if _, orphaned := scheduler.orphaned[removedRequirement]; !orphaned {
		t.Fatal("removed requirement did not become orphaned")
	}
	if scheduler.requirementRefs[sharedRequirement] != 1 ||
		scheduler.requirementRefs[addedRequirement] != 1 {
		t.Fatal("replacement did not retain the shared and added requirements")
	}
}
