package profile

import "fmt"

// ProductSurface is the second identity domain of the root contract:
// semantic product roots bound to current implementations. It is
// disjoint from the source universe — a ProductRootID is a stable
// TSTS-owned semantic identity, never a Go package or a file path, and
// domain membership is decided by these tagged records, never by
// string spelling.
type ProductSurface struct {
	Roots []ProductRoot `json:"roots"`
}

// ProductRoot is one tagged semantic root record.
//
// The public-api-set kind roots every exported declaration of every
// selected source package through its selector and carries no
// bindings. Every other kind is a concrete entry and must bind to at
// least one current ImplementationID; a binding-less concrete root is
// invalid, and reachability alone can never narrow the public API
// (root-contract.md, API Narrowing).
type ProductRoot struct {
	Kind string `json:"kind"` // public-api-set | assembly-entry | extension-entry | external-call-in | selected-test-entry | registration-entry
	ID   string `json:"id"`
	// Selector is required for public-api-set and must be
	// all-selected-exports; forbidden otherwise.
	Selector string `json:"selector,omitempty"`
	// Bindings are current ImplementationID values (ADR-0010 grammar),
	// required for every concrete kind; forbidden for public-api-set.
	Bindings []string `json:"bindings,omitempty"`
	Decision string   `json:"decision"`
	Reason   string   `json:"reason"`
}

var productRootKinds = map[string]bool{
	"public-api-set":      true,
	"assembly-entry":      true,
	"extension-entry":     true,
	"external-call-in":    true,
	"selected-test-entry": true,
	"registration-entry":  true,
}

// Validate enforces the tagged product-root contract.
func (s *ProductSurface) Validate() error {
	seen := map[string]bool{}
	publicAPI := 0
	for _, root := range s.Roots {
		if root.ID == "" || root.Decision == "" || root.Reason == "" {
			return fmt.Errorf("product root %q: id, decision, and reason are required", root.ID)
		}
		if seen[root.ID] {
			return fmt.Errorf("duplicate product root id %s", root.ID)
		}
		seen[root.ID] = true
		if !productRootKinds[root.Kind] {
			return fmt.Errorf("product root %s: unknown kind %q", root.ID, root.Kind)
		}
		if root.Kind == "public-api-set" {
			publicAPI++
			if root.Selector != "all-selected-exports" {
				return fmt.Errorf("product root %s: public-api-set requires selector all-selected-exports", root.ID)
			}
			if len(root.Bindings) != 0 {
				return fmt.Errorf("product root %s: public-api-set carries no bindings", root.ID)
			}
			continue
		}
		if root.Selector != "" {
			return fmt.Errorf("product root %s: selector is valid only on public-api-set", root.ID)
		}
		if len(root.Bindings) == 0 {
			return fmt.Errorf("product root %s (%s): a concrete root must bind to at least one current ImplementationID", root.ID, root.Kind)
		}
		for _, binding := range root.Bindings {
			if binding == "" {
				return fmt.Errorf("product root %s: empty binding", root.ID)
			}
		}
	}
	if publicAPI == 0 {
		return fmt.Errorf("product surface has no public-api-set root: the wide selected-export API is the mandatory initial policy")
	}
	return nil
}
