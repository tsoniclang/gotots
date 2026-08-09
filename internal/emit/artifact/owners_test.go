package artifact

import (
	"bytes"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestObservableContractCopiesFacetsAndOmitsImplementation(t *testing.T) {
	selected := artifactTestObject("Selected", 20)
	graph := NewGraph(compareArtifactTestObjects)
	contract := artifactTestContract(t, map[api.ArtifactFacet]string{
		api.ArtifactFacetCallableSignature: "signature",
		api.ArtifactFacetImplementation:    "body",
	})
	commitArtifactTestRevision(t, graph, selected, contract)

	observable, err := graph.ObservableContract(artifactTestOwner(selected))
	if err != nil {
		t.Fatal(err)
	}
	signature, present := observable.facet(api.ArtifactFacetCallableSignature)
	if !present || !bytes.Equal(signature, []byte("signature")) {
		t.Fatalf("observable signature = %q, %t", signature, present)
	}
	if _, present := observable.facet(api.ArtifactFacetImplementation); present {
		t.Fatal("observable contract retained implementation")
	}
	signature[0] = 'X'
	again, err := graph.ObservableContract(artifactTestOwner(selected))
	if err != nil {
		t.Fatal(err)
	}
	againSignature, _ := again.facet(api.ArtifactFacetCallableSignature)
	if !bytes.Equal(againSignature, []byte("signature")) {
		t.Fatalf("observable contract aliases graph storage: %q", againSignature)
	}
}

func TestObservableContractRejectsUnpublishedOwner(t *testing.T) {
	graph := NewGraph(compareArtifactTestObjects)
	missing := artifactTestObject("Missing", 10)
	if _, err := graph.ObservableContract(artifactTestOwner(missing)); err == nil {
		t.Fatal("unpublished observable contract passed")
	}
}
