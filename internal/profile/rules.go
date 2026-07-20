// Schema-2 source-universe contract: typed package rules with explicit
// override edges. Classification is total (every in-checkout package
// matches exactly one winning rule), overlap resolves ONLY through
// declared overrides, and neither array order nor prefix length has any
// authority. See .analysis root-contract.md (numbered-order steps 6-10).
package profile

import (
	"fmt"
	"sort"
	"strings"
)

// PackageDisposition is the closed source-universe verdict set.
type PackageDisposition string

const (
	DispositionSelected       PackageDisposition = "selected"
	DispositionTestOnly       PackageDisposition = "test-only"
	DispositionOutside        PackageDisposition = "outside-universe"
	DispositionTooling        PackageDisposition = "tooling"
	DispositionPolicyExcluded PackageDisposition = "product-policy-excluded"
)

// PackageSelector matches one package or one subtree at path-segment
// boundaries (internal/ast matches internal/ast and internal/ast/...,
// never internal/astnav).
type PackageSelector struct {
	Kind string `json:"kind"` // exact | subtree
	// Package/Root is the module-relative PackageRoot for the kind.
	Package string `json:"package,omitempty"`
	Root    string `json:"root,omitempty"`
}

// PackageRule is one typed disposition rule.
type PackageRule struct {
	ID          string             `json:"id"`
	Disposition PackageDisposition `json:"disposition"`
	Selectors   []PackageSelector  `json:"selectors"`
	// Overrides names every BROADER rule this rule's selectors carve out
	// of. An override is valid only when each of this rule's selectors is
	// strictly contained by some selector of the named rule.
	Overrides []string `json:"overrides"`
	Category  string   `json:"category"`
	Decision  string   `json:"decision"`
	Reason    string   `json:"reason"`
}

// DriftProbe names an evidence-only closure probe (never a root).
type DriftProbe struct {
	Kind string `json:"kind"` // upstream-main-closures
	Root string `json:"root"`
}

// SourceUniverse is the schema-2 source classification contract.
type SourceUniverse struct {
	PackageRules []PackageRule `json:"packageRules"`
	DriftProbes  []DriftProbe  `json:"driftProbes,omitempty"`
}

// selectorMatches reports whether one selector matches a module-relative
// package path, at segment boundaries.
func selectorMatches(s PackageSelector, pkg string) bool {
	switch s.Kind {
	case "exact":
		return pkg == s.Package
	case "subtree":
		return pkg == s.Root || strings.HasPrefix(pkg, s.Root+"/")
	}
	return false
}

// selectorRoot is the selector's anchor path (for containment checks).
func selectorRoot(s PackageSelector) string {
	if s.Kind == "exact" {
		return s.Package
	}
	return s.Root
}

// selectorContains reports whether outer strictly contains inner:
// every package inner can match is matched by outer, and inner is
// narrower than outer.
func selectorContains(outer, inner PackageSelector) bool {
	innerRoot := selectorRoot(inner)
	if !selectorMatches(outer, innerRoot) {
		return false
	}
	if outer.Kind == "exact" {
		// An exact selector contains only itself; "strictly" excludes it.
		return false
	}
	return innerRoot != selectorRoot(outer) || inner.Kind == "exact"
}

// Validate checks the structural contract: selector forms, override
// containment, override-target existence, and acyclicity.
func (u *SourceUniverse) Validate() error {
	byID := map[string]*PackageRule{}
	for i := range u.PackageRules {
		rule := &u.PackageRules[i]
		if rule.ID == "" || rule.Decision == "" || rule.Reason == "" || rule.Category == "" {
			return fmt.Errorf("package rule %q: id, category, decision, and reason are all required", rule.ID)
		}
		switch rule.Disposition {
		case DispositionSelected, DispositionTestOnly, DispositionOutside, DispositionTooling, DispositionPolicyExcluded:
		default:
			return fmt.Errorf("package rule %s: unknown disposition %q", rule.ID, rule.Disposition)
		}
		if byID[rule.ID] != nil {
			return fmt.Errorf("duplicate package rule id %s", rule.ID)
		}
		if len(rule.Selectors) == 0 {
			return fmt.Errorf("package rule %s: at least one selector is required", rule.ID)
		}
		for _, s := range rule.Selectors {
			// The selector is a strict field union: an exact selector
			// carries ONLY package, a subtree selector ONLY root. A
			// populated unused field is a silently ignored intent —
			// rejected, never tolerated.
			switch s.Kind {
			case "exact":
				if s.Package == "" || s.Root != "" {
					return fmt.Errorf("package rule %s: exact selector must set package and only package: %+v", rule.ID, s)
				}
			case "subtree":
				if s.Root == "" || s.Package != "" {
					return fmt.Errorf("package rule %s: subtree selector must set root and only root: %+v", rule.ID, s)
				}
			default:
				return fmt.Errorf("package rule %s: invalid selector kind %q", rule.ID, s.Kind)
			}
			root := selectorRoot(s)
			if strings.HasPrefix(root, "/") || strings.HasPrefix(root, "./") || strings.Contains(root, "..") {
				return fmt.Errorf("package rule %s: invalid selector root %q", rule.ID, root)
			}
		}
		byID[rule.ID] = rule
	}
	for i := range u.PackageRules {
		rule := &u.PackageRules[i]
		for _, target := range rule.Overrides {
			outer := byID[target]
			if outer == nil {
				return fmt.Errorf("package rule %s overrides unknown rule %s", rule.ID, target)
			}
			// Every selector of this rule that OVERLAPS the overridden
			// rule must be strictly contained by it; selectors unrelated
			// to the overridden rule are not part of this edge. The edge
			// itself must carve out at least one contained selector.
			carved := false
			for _, inner := range rule.Selectors {
				overlaps, contained := false, false
				for _, outerSel := range outer.Selectors {
					if selectorMatches(outerSel, selectorRoot(inner)) || selectorMatches(inner, selectorRoot(outerSel)) {
						overlaps = true
					}
					if selectorContains(outerSel, inner) {
						contained = true
					}
				}
				if overlaps && !contained {
					return fmt.Errorf("package rule %s: selector %s overlaps overridden rule %s without strict containment", rule.ID, selectorRoot(inner), target)
				}
				if contained {
					carved = true
				}
			}
			if !carved {
				return fmt.Errorf("package rule %s: override of %s carves out nothing (no contained selector)", rule.ID, target)
			}
		}
	}
	// Override acyclicity over the rule graph.
	state := map[string]int{}
	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("override cycle through rule %s", id)
		case 2:
			return nil
		}
		state[id] = 1
		for _, target := range byID[id].Overrides {
			if err := visit(target); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

// Classify resolves one module-relative package path to its winning
// rule. Errors are total: no match, or an unresolved overlap (two
// applicable rules with no declared override path between them).
func (u *SourceUniverse) Classify(pkg string) (*PackageRule, error) {
	var applicable []*PackageRule
	for i := range u.PackageRules {
		rule := &u.PackageRules[i]
		for _, s := range rule.Selectors {
			if selectorMatches(s, pkg) {
				applicable = append(applicable, rule)
				break
			}
		}
	}
	if len(applicable) == 0 {
		return nil, fmt.Errorf("package %s matches no rule (no implicit disposition exists)", pkg)
	}
	if len(applicable) == 1 {
		return applicable[0], nil
	}
	// Overlap: the winner is the applicable rule that (transitively)
	// overrides every other applicable rule; anything else is ambiguous.
	overridesAll := func(candidate *PackageRule) bool {
		reach := map[string]bool{}
		var walk func(id string)
		walk = func(id string) {
			for i := range u.PackageRules {
				if u.PackageRules[i].ID == id {
					for _, t := range u.PackageRules[i].Overrides {
						if !reach[t] {
							reach[t] = true
							walk(t)
						}
					}
				}
			}
		}
		walk(candidate.ID)
		for _, other := range applicable {
			if other.ID != candidate.ID && !reach[other.ID] {
				return false
			}
		}
		return true
	}
	var winners []*PackageRule
	for _, candidate := range applicable {
		if overridesAll(candidate) {
			winners = append(winners, candidate)
		}
	}
	if len(winners) != 1 {
		ids := make([]string, 0, len(applicable))
		for _, r := range applicable {
			ids = append(ids, r.ID)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("package %s: ambiguous rules %v (array order and prefix length have no authority; declare an override)", pkg, ids)
	}
	return winners[0], nil
}
