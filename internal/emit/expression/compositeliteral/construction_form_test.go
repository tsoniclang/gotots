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
	if got := constructionFormForReference(source, false); got != constructionFormDirectPositional {
		t.Fatalf("source construction form = %d, want direct positional", got)
	}
	if got := constructionFormForReference(source, true); got != constructionFormStorageObject {
		t.Fatalf("source storage form = %d, want storage object", got)
	}
	if got := constructionFormForReference(provider, false); got != constructionFormProviderFacet {
		t.Fatalf("provider construction form = %d, want certified positional facet", got)
	}
	if got := constructionFormForReference(provider, true); got != constructionFormProviderFacet {
		t.Fatalf("provider storage form = %d, want certified positional facet", got)
	}
}
