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
		key := canonicalTypeID(t)
		if seen[key] {
			return nil
		}
		seen[key] = true
		tokens = append(tokens, rtti)
		return nil
	}
	universe := append(append([]*types.TypeName{}, b.unit.ConcreteTypes()...), b.unit.ExternConcreteTypes()...)
	for _, name := range universe {
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
	key := canonicalIfaceID(iface)
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
		member := IfaceMember{
			K:   k,
			Pkg: obj.Pkg().Path(), Type: obj.Name(), Pointer: pointer,
			Struct: isStruct,
			Extern: !b.unit.Owns(obj.Pkg().Path()),
		}
		if member.Extern && !isStruct {
			// An external named type over a BASIC underlying boxes its
			// value carrier, not a branded handle.
			if basic, isBasic := named.Underlying().(*types.Basic); isBasic {
				member.ExternCarrier = basicCarrier(basic)
			}
		}
		members = append(members, member)
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
	// External implementers: the union member's vtable is built inline
	// over stub exports at box sites; each interface method becomes a
	// stub obligation so the adapters and exports exist.
	for _, name := range b.unit.ExternConcreteTypes() {
		named, ok := name.Type().(*types.Named)
		if !ok || (named.TypeParams() != nil && named.TypeParams().Len() > 0) {
			continue
		}
		register := func(pointer bool) {
			var t types.Type = named
			if pointer {
				t = types.NewPointer(named)
			}
			set := types.NewMethodSet(t)
			for i := range iface.NumMethods() {
				method := iface.Method(i)
				selection := lookupSelection(set, named.Obj().Pkg(), method.Name())
				if selection == nil {
					continue
				}
				fn := selection.Obj().(*types.Func)
				sig := fn.Type().(*types.Signature)
				if (sig.TypeParams() != nil && sig.TypeParams().Len() > 0) || SignatureMentionsTypeParam(sig) {
					continue // no single exact adapter
				}
				b.unit.AddExternalMethod(named.Obj().Pkg().Path(), named.Obj().Name(), fn)
			}
		}
		if types.Implements(named, iface) {
			if err := add(named, false); err != nil {
				return nil, err
			}
			register(false)
		}
		if types.Implements(types.NewPointer(named), iface) {
			if err := add(named, true); err != nil {
				return nil, err
			}
			register(true)
		}
	}
	sort.Slice(members, func(i, j int) bool { return members[i].K < members[j].K })
	b.unit.SetIfaceMemberCache(key, members)
	return members, nil
}

// basicCarrier spells the TS value carrier of a basic kind.
func basicCarrier(basic *types.Basic) string {
	switch {
	case basic.Info()&types.IsBoolean != 0:
		return "boolean"
	case basic.Info()&types.IsString != 0:
		return "string"
	case basic.Kind() == types.Int64 || basic.Kind() == types.Int ||
		basic.Kind() == types.Uint64 || basic.Kind() == types.Uint || basic.Kind() == types.Uintptr:
		return "bigint"
	case basic.Info()&types.IsNumeric != 0:
		return "number"
	}
	return ""
}

// SignatureMentionsTypeParam reports whether a signature references any
// type parameter.
func SignatureMentionsTypeParam(sig *types.Signature) bool {
	var mentions func(types.Type, map[types.Type]bool) bool
	mentions = func(t types.Type, seen map[types.Type]bool) bool {
		if t == nil || seen[t] {
			return false
		}
		seen[t] = true
		switch u := t.(type) {
		case *types.TypeParam:
			return true
		case *types.Named:
			if args := u.TypeArgs(); args != nil {
				for i := range args.Len() {
					if mentions(args.At(i), seen) {
						return true
					}
				}
			}
		case *types.Pointer:
			return mentions(u.Elem(), seen)
		case *types.Slice:
			return mentions(u.Elem(), seen)
		case *types.Array:
			return mentions(u.Elem(), seen)
		case *types.Map:
			return mentions(u.Key(), seen) || mentions(u.Elem(), seen)
		case *types.Chan:
			return mentions(u.Elem(), seen)
		case *types.Signature:
			for i := range u.Params().Len() {
				if mentions(u.Params().At(i).Type(), seen) {
					return true
				}
			}
			for i := range u.Results().Len() {
				if mentions(u.Results().At(i).Type(), seen) {
					return true
				}
			}
		}
		return false
	}
	seen := map[types.Type]bool{}
	for i := range sig.Params().Len() {
		if mentions(sig.Params().At(i).Type(), seen) {
			return true
		}
	}
	for i := range sig.Results().Len() {
		if mentions(sig.Results().At(i).Type(), seen) {
			return true
		}
	}
	return false
}

// registerBoxedExtern records the external named component of a boxed
// value's type in the dynamic-type universe.
func (b *builder) registerBoxedExtern(source types.Type) {
	t := types.Unalias(source)
	if pointer, ok := t.(*types.Pointer); ok {
		t = types.Unalias(pointer.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil || b.unit.Owns(named.Obj().Pkg().Path()) {
		return
	}
	if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
		return
	}
	if _, isIface := named.Underlying().(*types.Interface); isIface {
		return
	}
	b.unit.AddExternConcrete(named.Obj())
}

// mentionsTypeParamType reports whether a Go type references any type
// parameter.
func mentionsTypeParamType(t types.Type) bool {
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", t)), types.NewTuple(), false)
	return SignatureMentionsTypeParam(sig)
}
