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
// each once during the pre-pass). It is a construction defect to add a
// universe type after the universe is sealed — a late implementer could
// stale an already-cached union — so this panics past finalization.
func (s Scope) AddConcreteType(name *types.TypeName) {
	if *s.universeSealed {
		panic("ir: AddConcreteType after the dynamic-type universe was sealed")
	}
	*s.concreteTypes = append(*s.concreteTypes, name)
}

// ConcreteTypes returns the whole-unit named-type universe.
func (s Scope) ConcreteTypes() []*types.TypeName { return *s.concreteTypes }

// AddExternConcrete records one referenced external named type in the
// dynamic-type universe. It is called only from the whole-unit pre-pass
// freeze, which sorts and de-duplicates its candidates by canonical
// identity, so the universe is sealed once before any interface union is
// resolved; the object-identity guard keeps the contract idempotent
// regardless. Adding an external universe type after sealing is a
// construction defect and panics.
func (s Scope) AddExternConcrete(name *types.TypeName) {
	if *s.universeSealed {
		panic("ir: AddExternConcrete after the dynamic-type universe was sealed")
	}
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
	T  Type
	Eq *EqPlan
}

// AddBoxedComposite records one composite type boxed into an interface,
// with the recursive equality plan its empty-interface comparison uses.
// This is a BUILD-PHASE observation (a box site), so it requires the sealed
// universe — being observed before the pre-pass completes is a construction
// defect. It never mutates the sealed named-type universe and is read only
// at emit, so it feeds no cached union.
// AddValueBoxedNamed logs one named type's VALUE-form boxing.
func (s Scope) AddValueBoxedNamed(k string) { s.valueBoxedNamed[k] = true }

// ValueBoxedNamed reports whether a named type's value form is boxed
// anywhere in the unit (complete only after every body has built).
func (s Scope) ValueBoxedNamed(k string) bool { return s.valueBoxedNamed[k] }

func (s Scope) AddBoxedComposite(canon string, t Type, eq *EqPlan) {
	if !*s.universeSealed {
		panic("ir: AddBoxedComposite before the dynamic-type universe was sealed")
	}
	if _, has := s.boxedComposites[canon]; !has {
		s.boxedComposites[canon] = &boxedComposite{T: t, Eq: eq}
	}
}

// BoxedCompositeEntry is one boxed composite's exact payload plus its
// equality plan.
type BoxedCompositeEntry struct {
	Canon string
	T     Type
	Eq    *EqPlan
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
		out = append(out, BoxedCompositeEntry{Canon: id, T: c.T, Eq: c.Eq})
	}
	return out
}

// IfaceMemberCache returns a cached implementer union.
func (s Scope) IfaceMemberCache(key string) ([]IfaceMember, bool) {
	members, ok := s.ifaceMembers[key]
	return members, ok
}

// SetIfaceMemberCache stores one interface identity's implementer union.
// A union is a snapshot of the sealed dynamic-type universe: caching one
// before the universe is sealed would capture an incomplete set, so this
// panics if the universe is not yet finalized.
func (s Scope) SetIfaceMemberCache(key string, members []IfaceMember) {
	if !*s.universeSealed {
		panic("ir: interface union resolved before the dynamic-type universe was sealed")
	}
	s.ifaceMembers[key] = members
}
