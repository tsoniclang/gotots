package artifact

import (
	"bytes"
	"errors"
	"fmt"
	"go/token"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestArtifactGraphHistoryRetainsExactChangedRegions(t *testing.T) {
	owner := artifactTestObject("GrowingClass", 10)
	graph := NewGraph(compareArtifactTestObjects)
	const revisions = 256
	values := make([]string, 0, revisions)
	for revision := range revisions {
		value := "class{" + strings.Repeat("member;", revision+1) + "}"
		values = append(values, value)
		commitArtifactTestRevision(t, graph, owner, artifactCallable(value))
	}
	record := graph.records[artifactTestOwner(owner)]
	if got := record.history.retainedPayloadBytes(); got != 0 {
		t.Fatalf(
			"append-only history retained %d payload bytes, want zero exact changed-region bytes",
			got,
		)
	}
	err := graph.Commit(
		artifactTestOwner(owner),
		artifactCallable(values[revisions/2]),
		nil,
	)
	var convergenceError *ArtifactConvergenceError
	if !errors.As(err, &convergenceError) {
		t.Fatalf("lossless-history oscillation error = %#v", err)
	}
}

func TestContractReverseDeltaRestoresEveryFacetChangeClass(t *testing.T) {
	previous := artifactTestContract(t, map[api.ArtifactFacet]string{
		api.ArtifactFacetCallableSignature: "prefix-old-suffix",
		api.ArtifactFacetStaticSurface:     "removed",
		api.ArtifactFacetImplementation:    "unchanged",
	})
	next := artifactTestContract(t, map[api.ArtifactFacet]string{
		api.ArtifactFacetCallableSignature: "prefix-new-suffix",
		api.ArtifactFacetValueSurface:      "added",
		api.ArtifactFacetImplementation:    "unchanged",
	})
	restored := reverseContractDelta(previous, next).restore(next)
	if !equalArtifactContracts(restored, previous) {
		t.Fatal("lossless reverse delta did not restore add/remove/replace facets")
	}
}

func TestContractHistoryCompressesLargeExactChangedRegions(t *testing.T) {
	previous := artifactTestContractBytes(
		t,
		api.ArtifactFacetImplementation,
		bytes.Repeat([]byte("previous-method-body;"), 1<<15),
	)
	next := artifactTestContractBytes(
		t,
		api.ArtifactFacetImplementation,
		bytes.Repeat([]byte("replacement-method-body;"), 1<<15),
	)
	delta := reverseContractDelta(previous, next)
	payload := delta.facets[api.ArtifactFacetImplementation].previousMiddle
	if len(payload.data)*8 >= payload.rawLength {
		t.Fatalf(
			"exact history payload compressed from %d to %d bytes; expected at least 8x",
			payload.rawLength,
			len(payload.data),
		)
	}
	if restored := delta.restore(next); !equalArtifactContracts(
		restored,
		previous,
	) {
		t.Fatal("compressed history delta did not restore exact canonical bytes")
	}
}

func TestArtifactGraphFingerprintNeverEstablishesEquality(t *testing.T) {
	owner := artifactTestObject("CollisionFoil", 10)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(t, graph, owner, artifactCallable("first"))
	candidate := artifactCallable("different")
	record := graph.records[artifactTestOwner(owner)]
	record.history.entries[0].fingerprint = fingerprintContract(candidate)
	if err := graph.Commit(
		artifactTestOwner(owner),
		candidate,
		nil,
	); err != nil {
		t.Fatalf("fingerprint collision established false equality: %v", err)
	}
}

func TestArtifactGraphDirtySelectionIsLogarithmicAndStable(t *testing.T) {
	const consumerCount = 1024
	comparisons := 0
	graph := NewGraph(func(left api.ArtifactOwner, right api.ArtifactOwner) int {
		comparisons++
		return compareArtifactTestObjects(left, right)
	})
	provider := artifactTestObject("Provider", 1)
	commitArtifactTestRevision(t, graph, provider, artifactCallable("before"))
	for index := range consumerCount {
		consumer := artifactTestObject(
			fmt.Sprintf("Consumer%04d", index),
			token.Pos(index+2),
		)
		commitArtifactTestRevision(
			t,
			graph,
			consumer,
			artifactCallable("consumer"),
			artifactTestDependency(
				t,
				provider,
				api.ArtifactFacetCallableSignature,
			),
		)
	}
	comparisons = 0
	commitArtifactTestRevision(t, graph, provider, artifactCallable("after"))
	batch := graph.DirtyBatch()
	if len(batch) != consumerCount {
		t.Fatalf("dirty batch = %d consumers, want %d", len(batch), consumerCount)
	}
	var previous api.ArtifactOwner
	for index, owner := range batch {
		if index != 0 && compareArtifactTestObjects(previous, owner) >= 0 {
			t.Fatalf(
				"dirty order = %q then %q",
				previous.Name(),
				owner.Name(),
			)
		}
		previous = owner
		graph.DiscardDirty(owner)
	}
	if graph.HasPending() {
		t.Fatal("dirty queue retained work after complete drain")
	}
	if comparisons >= consumerCount*64 {
		t.Fatalf(
			"dirty queue used %d comparisons for %d consumers; quadratic scan restored",
			comparisons,
			consumerCount,
		)
	}
}

func TestArtifactGraphDirtyBatchSettlesProvidersBeforeConsumers(
	t *testing.T,
) {
	consumer := artifactTestObject("Consumer", 10)
	provider := artifactTestObject("Provider", 20)
	upstream := artifactTestObject("Upstream", 30)
	graph := NewGraph(compareArtifactTestObjects)
	commitArtifactTestRevision(t, graph, upstream, artifactCallable("upstream-a"))
	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactCallable("provider-a"),
		artifactTestDependency(
			t,
			upstream,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		consumer,
		artifactCallable("consumer"),
		artifactTestDependency(
			t,
			provider,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(
		t,
		graph,
		provider,
		artifactCallable("provider-b"),
		artifactTestDependency(
			t,
			upstream,
			api.ArtifactFacetCallableSignature,
		),
	)
	commitArtifactTestRevision(t, graph, upstream, artifactCallable("upstream-b"))

	batch := graph.DirtyBatch()
	want := []api.ArtifactOwner{
		artifactTestOwner(provider),
		artifactTestOwner(consumer),
	}
	if len(batch) != len(want) {
		t.Fatalf("dirty batch = %v, want %v", batch, want)
	}
	for index := range want {
		if batch[index] != want[index] {
			t.Fatalf("dirty batch = %v, want provider before consumer", batch)
		}
	}
	if !graph.HasPending() {
		t.Fatal("dirty batch inspection consumed work before reconstruction")
	}
}
