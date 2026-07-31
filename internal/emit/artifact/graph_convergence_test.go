package artifact

import (
	"errors"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestArtifactGraphConvergesCyclesAndRejectsOscillation(t *testing.T) {
	first := artifactTestObject("First", 10)
	second := artifactTestObject("Second", 20)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(
		t,
		graph,
		first,
		artifactCallable("first-a"),
		artifactTestDependency(
			t,
			second,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		second,
		artifactCallable("second-a"),
		artifactTestDependency(
			t,
			first,
			api.ArtifactFacetCallableSignature,
		),
	)
	dirty, ok := graph.NextDirty()
	if !ok || dirty != artifactTestOwner(first) {
		t.Fatalf(
			"first-publication dirty = %v, %t; want first",
			dirty,
			ok,
		)
	}
	commitArtifactTestRevision(
		t,
		graph,
		first,
		artifactCallable("first-a"),
		artifactTestDependency(
			t,
			second,
			api.ArtifactFacetCallableSignature,
		),
	)
	if graph.HasPending() {
		t.Fatal("first-publication reconstruction did not converge")
	}

	commitArtifactTestRevision(t, graph, first, artifactCallable("first-b"))
	dirty, ok = graph.NextDirty()
	if !ok || dirty != artifactTestOwner(second) {
		t.Fatalf("cycle dirty = %v, %t; want second", dirty, ok)
	}
	commitArtifactTestRevision(
		t,
		graph,
		second,
		artifactCallable("second-a"),
		artifactTestDependency(
			t,
			first,
			api.ArtifactFacetCallableSignature,
		),
	)
	if graph.HasPending() {
		t.Fatal("stable cycle did not converge")
	}

	err := graph.Commit(
		artifactTestOwner(first),
		artifactCallable("first-a"),
		nil,
	)
	var convergenceError *ArtifactConvergenceError
	if !errors.As(err, &convergenceError) ||
		convergenceError.Object != artifactTestOwner(first) ||
		len(convergenceError.Facets) != 1 ||
		convergenceError.Facets[0] != api.ArtifactFacetCallableSignature {
		t.Fatalf("oscillation error = %#v", err)
	}
}

func TestArtifactGraphAllowsOneRequirementRemovalReversion(
	t *testing.T,
) {
	object := artifactTestObject("Provider", 10)
	owner := artifactTestOwner(object)
	graph := NewGraph(compareArtifactTestObjects)
	if err := graph.Commit(owner, artifactCallable("base"), nil); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(owner, artifactCallable("profile"), nil); err != nil {
		t.Fatal(err)
	}
	if err := graph.CommitHistoricalReplacement(
		owner,
		artifactCallable("base"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	err := graph.Commit(owner, artifactCallable("profile"), nil)
	var convergenceError *ArtifactConvergenceError
	if !errors.As(err, &convergenceError) ||
		convergenceError.Object != owner {
		t.Fatalf("reintroduced removed requirement error = %#v", err)
	}
}

func TestRequirementRemovalAuthorityPropagatesToDirtyConsumers(
	t *testing.T,
) {
	provider := artifactTestObject("Provider", 10)
	consumer := artifactTestObject("Consumer", 20)
	providerOwner := artifactTestOwner(provider)
	consumerOwner := artifactTestOwner(consumer)
	providerDependency := artifactTestDependency(
		t,
		provider,
		api.ArtifactFacetCallableSignature,
	)
	graph := NewGraph(compareArtifactTestObjects)
	if err := graph.Commit(
		providerOwner,
		artifactCallable("base"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		consumerOwner,
		artifactCallable("consumer-base"),
		[]api.ArtifactDependency{providerDependency},
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		providerOwner,
		artifactCallable("profile"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		consumerOwner,
		artifactCallable("consumer-profile"),
		[]api.ArtifactDependency{providerDependency},
	); err != nil {
		t.Fatal(err)
	}
	graph.DiscardDirty(consumerOwner)

	if err := graph.CommitHistoricalReplacement(
		providerOwner,
		artifactCallable("base"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		consumerOwner,
		artifactCallable("consumer-base"),
		[]api.ArtifactDependency{providerDependency},
	); err != nil {
		t.Fatalf("removal-caused consumer reconstruction: %v", err)
	}
	graph.DiscardDirty(consumerOwner)

	err := graph.Commit(
		consumerOwner,
		artifactCallable("consumer-profile"),
		[]api.ArtifactDependency{providerDependency},
	)
	var convergenceError *ArtifactConvergenceError
	if !errors.As(err, &convergenceError) ||
		convergenceError.Object != consumerOwner {
		t.Fatalf("consumer reintroduction error = %#v", err)
	}
}

func TestRequirementRemovalAuthorityPropagatesTransitively(
	t *testing.T,
) {
	provider := artifactTestObject("Provider", 10)
	middle := artifactTestObject("Middle", 20)
	leaf := artifactTestObject("Leaf", 30)
	providerOwner := artifactTestOwner(provider)
	middleOwner := artifactTestOwner(middle)
	leafOwner := artifactTestOwner(leaf)
	providerDependency := artifactTestDependency(
		t,
		provider,
		api.ArtifactFacetCallableSignature,
	)
	middleDependency := artifactTestDependency(
		t,
		middle,
		api.ArtifactFacetCallableSignature,
	)
	graph := NewGraph(compareArtifactTestObjects)
	if err := graph.Commit(
		providerOwner,
		artifactCallable("provider-base"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		middleOwner,
		artifactCallable("middle-base"),
		[]api.ArtifactDependency{providerDependency},
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		leafOwner,
		artifactCallable("leaf-base"),
		[]api.ArtifactDependency{middleDependency},
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		providerOwner,
		artifactCallable("provider-profile"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		middleOwner,
		artifactCallable("middle-profile"),
		[]api.ArtifactDependency{providerDependency},
	); err != nil {
		t.Fatal(err)
	}
	graph.DiscardDirty(middleOwner)
	if err := graph.Commit(
		leafOwner,
		artifactCallable("leaf-profile"),
		[]api.ArtifactDependency{middleDependency},
	); err != nil {
		t.Fatal(err)
	}
	graph.DiscardDirty(leafOwner)

	if err := graph.CommitHistoricalReplacement(
		providerOwner,
		artifactCallable("provider-base"),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := graph.Commit(
		middleOwner,
		artifactCallable("middle-base"),
		[]api.ArtifactDependency{providerDependency},
	); err != nil {
		t.Fatalf("middle removal reconstruction: %v", err)
	}
	graph.DiscardDirty(middleOwner)
	if err := graph.Commit(
		leafOwner,
		artifactCallable("leaf-base"),
		[]api.ArtifactDependency{middleDependency},
	); err != nil {
		t.Fatalf("leaf removal reconstruction: %v", err)
	}
	graph.DiscardDirty(leafOwner)

	err := graph.Commit(
		leafOwner,
		artifactCallable("leaf-profile"),
		[]api.ArtifactDependency{middleDependency},
	)
	var convergenceError *ArtifactConvergenceError
	if !errors.As(err, &convergenceError) ||
		convergenceError.Object != leafOwner {
		t.Fatalf("leaf reintroduction error = %#v", err)
	}
}
