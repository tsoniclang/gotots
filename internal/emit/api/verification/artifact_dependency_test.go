package api_test

import (
	"errors"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestArtifactDependencyRequestIsClosedAndIdentityKeyed(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/dependency", "dependency")
	provider := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Provide",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	request, err := api.NewArtifactDependencyRequest(
		provider,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		t.Fatal(err)
	}
	if request.Kind() != api.RootRequestArtifactDependency {
		t.Fatalf("request kind = %v", request.Kind())
	}
	dependency, ok := request.ArtifactDependency()
	sourceProvider, sourceOwned := dependency.Provider().Source()
	if !ok ||
		!sourceOwned ||
		sourceProvider != provider ||
		dependency.Facet() != api.ArtifactFacetCallableSignature {
		t.Fatalf("dependency = %#v, %t", dependency, ok)
	}

	for name, test := range map[string]struct {
		provider types.Object
		facet    api.ArtifactFacet
	}{
		"nil provider":  {facet: api.ArtifactFacetCallableSignature},
		"invalid facet": {provider: provider},
		"unknown facet": {provider: provider, facet: api.ArtifactFacet(255)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := api.NewArtifactDependencyRequest(
				test.provider,
				test.facet,
			)
			var requestError *api.RootRequestError
			if !errors.As(err, &requestError) {
				t.Fatalf("error = %#v, want RootRequestError", err)
			}
		})
	}
}

func TestArtifactFacetIDsArePinned(t *testing.T) {
	actual := []api.ArtifactFacet{
		api.ArtifactFacetCallableSignature,
		api.ArtifactFacetConstructorSurface,
		api.ArtifactFacetInstanceTypeSurface,
		api.ArtifactFacetStaticSurface,
		api.ArtifactFacetValueSurface,
	}
	expected := []api.ArtifactFacet{1, 2, 3, 4, 5}
	for index := range expected {
		if actual[index] != expected[index] || !actual[index].Valid() {
			t.Fatalf("facet %d = %d", index, actual[index])
		}
	}
	if api.ArtifactFacetInvalid.Valid() || api.ArtifactFacet(6).Valid() {
		t.Fatal("artifact facet domain is not closed")
	}
}
