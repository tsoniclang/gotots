package emit

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/ir"
)

// IfaceArtifacts is the unit-shared canonical owner of interface union
// artifacts (ADR-0008): each closed-union TYPE, its equality function,
// and its map-key encoder is registered ONCE by whichever module first
// references it, and emitted ONCE in the canonical interfaces module.
// Every consumer references the canonical symbols through the goifc$
// namespace, so a union declared 48 times across modules becomes one
// definition and 48 type-only/value references.
type IfaceArtifacts struct {
	// order preserves first-registration order for deterministic output.
	order []string
	types map[string]ir.Type
	// keyRequired marks unions whose $key encoder some consumer needs.
	keyRequired map[string]bool
	// identity guards against a digest collision between distinct
	// interface identities sharing one alias name.
	identity map[string]string
}

func NewIfaceArtifacts() *IfaceArtifacts {
	return &IfaceArtifacts{
		types:       map[string]ir.Type{},
		keyRequired: map[string]bool{},
		identity:    map[string]string{},
	}
}

// Register records one canonical union by alias name and returns
// whether the name/identity pair is consistent. A digest collision
// between distinct identities fails closed.
func (a *IfaceArtifacts) Register(name, identity string, t ir.Type) error {
	if prior, exists := a.identity[name]; exists {
		if prior != identity {
			return &aliasCollisionError{name: name, prior: prior, next: identity}
		}
		return nil
	}
	a.identity[name] = identity
	a.types[name] = t
	a.order = append(a.order, name)
	return nil
}

// RequireKey marks a union's key encoder as needed.
func (a *IfaceArtifacts) RequireKey(name string) {
	a.keyRequired[name] = true
}

// Names returns registered union names in first-registration order.
func (a *IfaceArtifacts) Names() []string { return a.order }

// Type returns a registered union's interface type.
func (a *IfaceArtifacts) Type(name string) ir.Type { return a.types[name] }

// KeyRequired reports whether a union's key encoder is needed anywhere.
func (a *IfaceArtifacts) KeyRequired(name string) bool { return a.keyRequired[name] }

// SortedNames returns the names sorted (for stable diagnostics).
func (a *IfaceArtifacts) SortedNames() []string {
	out := append([]string{}, a.order...)
	sort.Strings(out)
	return out
}

type aliasCollisionError struct{ name, prior, next string }

func (e *aliasCollisionError) Error() string {
	return "interface alias digest collision: " + e.name + " names both " + e.prior + " and " + e.next
}
