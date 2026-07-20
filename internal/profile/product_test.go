package profile

import (
	"strings"
	"testing"
)

func surfaceWith(roots ...ProductRoot) *ProductSurface {
	return &ProductSurface{Roots: roots}
}

func apiRoot() ProductRoot {
	return ProductRoot{Kind: "public-api-set", ID: "p.api", Selector: "all-selected-exports",
		Decision: "D", Reason: "R"}
}

func TestProductSurfaceRequiresPublicAPIRoot(t *testing.T) {
	if err := surfaceWith().Validate(); err == nil || !strings.Contains(err.Error(), "no public-api-set root") {
		t.Fatalf("empty surface must fail on the missing API root: %v", err)
	}
	if err := surfaceWith(apiRoot()).Validate(); err != nil {
		t.Fatal(err)
	}
}

// MUTATION: a concrete root without ImplementationID bindings fails.
func TestConcreteRootRequiresBindings(t *testing.T) {
	root := ProductRoot{Kind: "assembly-entry", ID: "p.cli", Decision: "D", Reason: "R"}
	if err := surfaceWith(apiRoot(), root).Validate(); err == nil || !strings.Contains(err.Error(), "must bind") {
		t.Fatalf("binding-less concrete root must fail: %v", err)
	}
	root.Bindings = []string{"pkg::func::Main/default"}
	if err := surfaceWith(apiRoot(), root).Validate(); err != nil {
		t.Fatal(err)
	}
}

// MUTATION: selector is exclusive to public-api-set; bindings are
// forbidden on it; unknown kinds and duplicate IDs fail.
func TestProductRootTagDiscipline(t *testing.T) {
	bad := []ProductRoot{
		{Kind: "assembly-entry", ID: "x", Selector: "all-selected-exports", Bindings: []string{"b"}, Decision: "D", Reason: "R"},
		{Kind: "public-api-set", ID: "x", Selector: "everything", Decision: "D", Reason: "R"},
		{Kind: "public-api-set", ID: "x", Selector: "all-selected-exports", Bindings: []string{"b"}, Decision: "D", Reason: "R"},
		{Kind: "cli-path", ID: "x", Bindings: []string{"b"}, Decision: "D", Reason: "R"},
	}
	for _, root := range bad {
		if err := surfaceWith(apiRoot(), root).Validate(); err == nil {
			t.Fatalf("root %+v must fail validation", root)
		}
	}
	dup := apiRoot()
	dup.ID = "p.api"
	if err := surfaceWith(apiRoot(), dup).Validate(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate id must fail: %v", err)
	}
}
