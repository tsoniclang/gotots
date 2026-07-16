// Closed-world interface dispatch resolution. Every interface method
// call resolves its complete reachable dynamic-type set from the unit's
// concrete-type universe: the closed set of named types (owned and
// referenced external) whose method set includes the interface's
// method. Dispatch lowers to an exhaustive token switch over that set —
// direct typed calls, never a name-selected member lookup. A dynamic
// type set that cannot be proven closed fails closed.
package ir

import (
	"go/types"
	"sort"
)

// lookupSelection returns the method selection named `name` in a method
// set, resolving exported methods without package qualification.
func lookupSelection(set *types.MethodSet, pkg *types.Package, name string) *types.Selection {
	if selection := set.Lookup(pkg, name); selection != nil {
		return selection
	}
	for i := range set.Len() {
		if set.At(i).Obj().Name() == name {
			return set.At(i)
		}
	}
	return nil
}

// resolveImplementerTokens computes the closed set of dynamic-type
// tokens in the unit universe that satisfy the target interface: the
// static membership set for an interface-to-interface assertion.
func (b *builder) resolveImplementerTokens(target *types.Interface, span Span) ([]RttiRef, error) {
	var tokens []RttiRef
	seen := map[string]bool{}
	add := func(t types.Type) error {
		rtti, err := b.rttiFor(t, span)
		if err != nil {
			return err
		}
		key := t.String()
		if seen[key] {
			return nil
		}
		seen[key] = true
		tokens = append(tokens, rtti)
		return nil
	}
	for _, name := range b.unit.ConcreteTypes() {
		named, ok := name.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, isIface := named.Underlying().(*types.Interface); isIface {
			continue
		}
		if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			continue
		}
		if types.Implements(named, target) {
			if err := add(named); err != nil {
				return nil, err
			}
		}
		if types.Implements(types.NewPointer(named), target) {
			if err := add(types.NewPointer(named)); err != nil {
				return nil, err
			}
		}
	}
	return tokens, nil
}

// ifaceMembers resolves the closed implementer union of one interface
// type from the whole-unit universe: only TYPE identities (spelling uses
// erased type-only imports), cached per canonical interface identity.
func (b *builder) ifaceMembers(iface *types.Interface, span Span) ([]IfaceMember, error) {
	key := types.TypeString(iface, func(p *types.Package) string { return p.Path() })
	if cached, ok := b.unit.IfaceMemberCache(key); ok {
		return cached, nil
	}
	var members []IfaceMember
	add := func(named *types.Named, pointer bool) error {
		obj := named.Obj()
		k := obj.Pkg().Path() + "." + obj.Name()
		if pointer {
			k = "*" + k
		}
		_, isStruct := named.Underlying().(*types.Struct)
		members = append(members, IfaceMember{
			K:   k,
			Pkg: obj.Pkg().Path(), Type: obj.Name(), Pointer: pointer,
			Struct: isStruct,
			Extern: !b.unit.Owns(obj.Pkg().Path()),
		})
		return nil
	}
	for _, name := range b.unit.ConcreteTypes() {
		named, ok := name.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, isIface := named.Underlying().(*types.Interface); isIface {
			continue
		}
		if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			continue
		}
		if types.Implements(named, iface) {
			if err := add(named, false); err != nil {
				return nil, err
			}
		}
		if types.Implements(types.NewPointer(named), iface) {
			if err := add(named, true); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].K < members[j].K })
	b.unit.SetIfaceMemberCache(key, members)
	return members, nil
}
