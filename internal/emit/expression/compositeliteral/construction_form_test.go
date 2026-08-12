package compositeliteral

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestCertifiedProviderBoundaryAloneSelectsPositionalConstruction(
	t *testing.T,
) {
	source, err := api.NewNameReference("Record")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := api.NewProviderQualifiedNameReference(
		"records__from_provider",
		"RecordOperations",
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := constructionFormForReference(source); got != constructionFormNamedObject {
		t.Fatalf("source construction form = %d, want named object", got)
	}
	if got := constructionFormForReference(provider); got != constructionFormProviderFacet {
		t.Fatalf("provider construction form = %d, want certified positional facet", got)
	}
}
