package ir

import (
	"go/types"
	"sort"
)

// This file holds the unit-wide dynamic-type universe of a Scope: the
// concrete-type set, the external-implementer set, the boxed-composite
// enumeration, and the resolved interface-union cache — the closed-world
// evidence the interface subsystem resolves closed unions over.

// AddConcreteType records one named type in the unit's dynamic-type
// universe (idempotent by object identity is not enforced; callers add
// each once during the pre-pass).
func (s Scope) AddConcreteType(name *types.TypeName) {
	*s.concreteTypes = append(*s.concreteTypes, name)
}

// ConcreteTypes returns the whole-unit named-type universe.
func (s Scope) ConcreteTypes() []*types.TypeName { return *s.concreteTypes }

// AddExternConcrete records one referenced external named type in the
// dynamic-type universe (idempotent: a type registered twice — e.g. boxed
// in two bodies, or pre-frozen then re-seen — appears once).
func (s Scope) AddExternConcrete(name *types.TypeName) {
	for _, existing := range *s.externConcrete {
		if existing == name {
			return
		}
	}
	*s.externConcrete = append(*s.externConcrete, name)
}

// ExternConcreteTypes returns the referenced external named types.
func (s Scope) ExternConcreteTypes() []*types.TypeName { return *s.externConcrete }

// boxedComposite records one composite boxed into an interface: its
// exact payload type and how two of its values compare, so the
// empty-interface equality narrows to an exact per-member operation.
type boxedComposite struct {
	T           Type
	EqMode      string
	ArrayElemEq string
}

// AddBoxedComposite records one composite type boxed into an interface,
// with the per-member equality mode its empty-interface comparison uses.
func (s Scope) AddBoxedComposite(canon string, t Type, eqMode, arrayElemEq string) {
	if _, has := s.boxedComposites[canon]; !has {
		s.boxedComposites[canon] = &boxedComposite{T: t, EqMode: eqMode, ArrayElemEq: arrayElemEq}
	}
}

// BoxedCompositeEntry is one boxed composite's exact payload plus its
// equality mode.
type BoxedCompositeEntry struct {
	Canon       string
	T           Type
	EqMode      string
	ArrayElemEq string
}

// BoxedComposites returns the boxed-composite enumeration sorted by id.
func (s Scope) BoxedComposites() []BoxedCompositeEntry {
	ids := make([]string, 0, len(s.boxedComposites))
	for id := range s.boxedComposites {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]BoxedCompositeEntry, 0, len(ids))
	for _, id := range ids {
		c := s.boxedComposites[id]
		out = append(out, BoxedCompositeEntry{Canon: id, T: c.T, EqMode: c.EqMode, ArrayElemEq: c.ArrayElemEq})
	}
	return out
}

// IfaceMemberCache returns a cached implementer union.
func (s Scope) IfaceMemberCache(key string) ([]IfaceMember, bool) {
	members, ok := s.ifaceMembers[key]
	return members, ok
}

// SetIfaceMemberCache stores one interface identity's implementer union.
func (s Scope) SetIfaceMemberCache(key string, members []IfaceMember) {
	s.ifaceMembers[key] = members
}
