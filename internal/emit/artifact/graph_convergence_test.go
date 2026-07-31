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
