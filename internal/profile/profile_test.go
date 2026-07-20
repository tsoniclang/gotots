package profile

import "testing"

func v2Profile() *Profile {
	return &Profile{
		SchemaVersion: 2,
		GoModule:      "example.com/mod",
		SourceUniverse: SourceUniverse{PackageRules: []PackageRule{
			{ID: "selected", Disposition: DispositionSelected,
				Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/ast"}},
				Overrides: []string{}, Category: "product-source", Decision: "D", Reason: "r"},
			{ID: "outside-lsp", Disposition: DispositionOutside,
				Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/lsp"}},
				Overrides: []string{}, Category: "lsp", Decision: "D", Reason: "r"},
			{ID: "tooling", Disposition: DispositionTooling,
				Selectors: []PackageSelector{{Kind: "subtree", Root: "_tools"}},
				Overrides: []string{}, Category: "build-tooling", Decision: "D", Reason: "r"},
		}},
	}
}

// Classify routes through the schema-2 contract: every disposition maps
// to its class, unmatched module packages are UNCLASSIFIED defects, and
// packages outside the module are external.
func TestClassifyV2(t *testing.T) {
	p := v2Profile()
	cases := []struct {
		pkg      string
		class    PackageClass
		category string
	}{
		{"example.com/mod/internal/ast", ClassOwned, "product-source"},
		{"example.com/mod/internal/ast/deep", ClassOwned, "product-source"},
		{"example.com/mod/internal/lsp", ClassOutsideUniverse, "lsp"},
		{"example.com/mod/_tools", ClassTooling, "build-tooling"},
		{"other.example.com/pkg", ClassExternal, ""},
	}
	for _, c := range cases {
		class, category := p.Classify(c.pkg)
		if class != c.class || category != c.category {
			t.Fatalf("%s: got (%s, %q), want (%s, %q)", c.pkg, class, category, c.class, c.category)
		}
	}
	if class, detail := p.Classify("example.com/mod/internal/unknown"); class != ClassUnclassified || detail == "" {
		t.Fatalf("unmatched module package must be unclassified with detail, got (%s, %q)", class, detail)
	}
}
