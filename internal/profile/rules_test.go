package profile

import (
	"strings"
	"testing"
)

func contractFixture() *SourceUniverse {
	return &SourceUniverse{PackageRules: []PackageRule{
		{ID: "selected", Disposition: DispositionSelected,
			Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/ast"}, {Kind: "subtree", Root: "internal/execute/incremental"}, {Kind: "subtree", Root: "internal/execute/tsc"}},
			Overrides: []string{"execute-driver"}, Category: "product-source", Decision: "TSTS-PRODUCT-SURFACE", Reason: "selected"},
		{ID: "execute-driver", Disposition: DispositionPolicyExcluded,
			Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/execute"}},
			Overrides: []string{}, Category: "upstream-driver", Decision: "ADR-0003", Reason: "TSTS owns its driver"},
		{ID: "test-support", Disposition: DispositionTestOnly,
			Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/testutil"}},
			Overrides: []string{}, Category: "portable-test-support", Decision: "TSTS-TEST-SCOPE", Reason: "test-only"},
		{ID: "editor-test-support", Disposition: DispositionOutside,
			Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/testutil/lsptestutil"}},
			Overrides: []string{"test-support"}, Category: "editor-service", Decision: "TSTS-SCOPE-EXCLUSION", Reason: "excluded family"},
	}}
}

func TestContractCases(t *testing.T) {
	u := contractFixture()
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := map[string]PackageDisposition{
		"internal/ast":                  DispositionSelected,
		"internal/ast/astnav":           DispositionSelected,
		"internal/execute":              DispositionPolicyExcluded,
		"internal/execute/build":        DispositionPolicyExcluded, // inherits ADR-0003; no redundant rule
		"internal/execute/incremental":  DispositionSelected,
		"internal/execute/tsc":          DispositionSelected,
		"internal/testutil":             DispositionTestOnly,
		"internal/testutil/lsptestutil": DispositionOutside,
	}
	for pkg, want := range cases {
		rule, err := u.Classify(pkg)
		if err != nil {
			t.Fatalf("%s: %v", pkg, err)
		}
		if rule.Disposition != want {
			t.Fatalf("%s: got %s (rule %s), want %s", pkg, rule.Disposition, rule.ID, want)
		}
	}
}

func TestSegmentBoundaryMatching(t *testing.T) {
	u := &SourceUniverse{PackageRules: []PackageRule{
		{ID: "ast", Disposition: DispositionSelected, Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/ast"}},
			Overrides: []string{}, Category: "c", Decision: "d", Reason: "r"},
	}}
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Classify("internal/astnav"); err == nil {
		t.Fatal("internal/astnav must NOT match the internal/ast subtree")
	}
}

func TestNoMatchIsError(t *testing.T) {
	u := contractFixture()
	if _, err := u.Classify("internal/unknown"); err == nil || !strings.Contains(err.Error(), "no rule") {
		t.Fatalf("expected total-classification error, got %v", err)
	}
}

// MUTATION: overlap without a declared override is ambiguous — array
// order and prefix length have no authority.
func TestUndeclaredOverlapIsAmbiguous(t *testing.T) {
	u := contractFixture()
	u.PackageRules = append(u.PackageRules, PackageRule{
		ID: "rogue", Disposition: DispositionTooling,
		Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/execute/incremental"}},
		Overrides: []string{}, Category: "c", Decision: "d", Reason: "r"})
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Classify("internal/execute/incremental"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

// MUTATION: an override whose selector is NOT strictly contained by the
// overridden rule is invalid.
func TestNonContainedOverrideRejected(t *testing.T) {
	u := contractFixture()
	u.PackageRules = append(u.PackageRules, PackageRule{
		ID: "bad", Disposition: DispositionSelected,
		Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/elsewhere"}},
		Overrides: []string{"execute-driver"}, Category: "c", Decision: "d", Reason: "r"})
	if err := u.Validate(); err == nil || !strings.Contains(err.Error(), "carves out nothing") {
		t.Fatalf("expected containment error, got %v", err)
	}
}

// MUTATION: attempted override cycles cannot validate. Strict
// containment makes a true cycle geometrically unconstructible — the
// overlapping-selector check rejects the construction before the
// acyclicity walker (which remains as defense in depth) can ever see
// a valid cycle.
func TestOverrideCycleRejected(t *testing.T) {
	u := &SourceUniverse{PackageRules: []PackageRule{
		{ID: "a", Disposition: DispositionSelected, Selectors: []PackageSelector{{Kind: "subtree", Root: "x/y"}},
			Overrides: []string{"b"}, Category: "c", Decision: "d", Reason: "r"},
		{ID: "b", Disposition: DispositionTooling, Selectors: []PackageSelector{{Kind: "subtree", Root: "x"}, {Kind: "subtree", Root: "x/y/z"}},
			Overrides: []string{"a"}, Category: "c", Decision: "d", Reason: "r"},
	}}
	if err := u.Validate(); err == nil {
		t.Fatal("cyclic override construction must not validate")
	}
}

func TestMissingProvenanceRejected(t *testing.T) {
	u := contractFixture()
	u.PackageRules[0].Decision = ""
	if err := u.Validate(); err == nil {
		t.Fatal("expected provenance error")
	}
}

// MUTATION: an exact selector never widens to a subtree load pattern,
// and the selector field union is strict.
func TestExactSelectorLoadPatternFidelity(t *testing.T) {
	u := &SourceUniverse{PackageRules: []PackageRule{
		{ID: "one-exact", Disposition: DispositionSelected,
			Selectors: []PackageSelector{{Kind: "exact", Package: "internal/one"}},
			Category:  "c", Decision: "d", Reason: "r"},
		{ID: "tree", Disposition: DispositionSelected,
			Selectors: []PackageSelector{{Kind: "subtree", Root: "internal/tree"}},
			Category:  "c", Decision: "d", Reason: "r"},
	}}
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	patterns := u.SelectedLoadPatterns()
	want := []string{"./internal/one", "./internal/tree/..."}
	if len(patterns) != 2 || patterns[0] != want[0] || patterns[1] != want[1] {
		t.Fatalf("patterns = %v; want %v (exact must NOT widen)", patterns, want)
	}
}

func TestSelectorFieldUnionIsStrict(t *testing.T) {
	cases := []PackageSelector{
		{Kind: "exact", Package: "a", Root: "b"}, // both set
		{Kind: "subtree", Root: "a", Package: "b"},
		{Kind: "exact", Root: "a"}, // wrong field
		{Kind: "subtree", Package: "a"},
	}
	for _, selector := range cases {
		u := &SourceUniverse{PackageRules: []PackageRule{
			{ID: "x", Disposition: DispositionSelected,
				Selectors: []PackageSelector{selector},
				Category:  "c", Decision: "d", Reason: "r"},
		}}
		if err := u.Validate(); err == nil {
			t.Fatalf("selector %+v must fail the field-union validation", selector)
		}
	}
}
