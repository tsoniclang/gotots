// Closed-world interface dispatch resolution. Every interface method
// call resolves its complete reachable dynamic-type set from the unit's
// concrete-type universe: the closed set of named types (owned and
// referenced external) whose method set includes the interface's
// method. Dispatch lowers to an exhaustive token switch over that set —
// direct typed calls, never a name-selected member lookup. A dynamic
// type set that cannot be proven closed fails closed.
package ir

import (
	"fmt"
	"go/types"
	"sort"
)

// lookupSelection returns the interface method's selection in a concrete
// type's method set by Go's own rule: exported names match regardless of
// package, unexported names match only within their declaring package.
// The pkg MUST be the INTERFACE METHOD's declaring package (not the
// concrete type's) — there is no bare-name fallback, which would match
// an unrelated same-spelled unexported method from another package.
func lookupSelection(set *types.MethodSet, pkg *types.Package, name string) *types.Selection {
	return set.Lookup(pkg, name)
}

// resolveImplementerTokens computes the closed set of dynamic-type tokens
// for an interface-to-interface assertion x.(Target): the universe types
// that satisfy the TARGET and are also possible dynamic types of the
// SOURCE (they implement it). Intersecting with the source keeps the
// assertion's literal checks within the source union's discriminants —
// enumerating a target implementer absent from the source would compare
// the source's k against a literal outside its type.
func (b *builder) resolveImplementerTokens(source, target *types.Interface, span Span) ([]RttiRef, error) {
	var tokens []RttiRef
	seen := map[string]bool{}
	add := func(t types.Type) error {
		rtti, err := b.rttiFor(t, span)
		if err != nil {
			return err
		}
		key, err := b.canonicalTypeID(t)
		if err != nil {
			return err
		}
		if seen[key] {
			return nil
		}
		seen[key] = true
		tokens = append(tokens, rtti)
		return nil
	}
	// A non-empty source restricts the candidates to its own implementers;
	// the empty interface admits every type, so it imposes no filter.
	inSource := func(t types.Type) bool {
		return source == nil || source.NumMethods() == 0 || types.Implements(t, source)
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
		if types.Implements(named, target) && inSource(named) {
			if err := add(named); err != nil {
				return nil, err
			}
		}
		if pointer := types.NewPointer(named); types.Implements(pointer, target) && inSource(pointer) {
			if err := add(pointer); err != nil {
				return nil, err
			}
		}
	}
	return tokens, nil
}

// ifaceMembers resolves the closed implementer union of one interface
// type from the whole-unit universe: only TYPE identities (spelling uses
// erased type-only imports), cached per canonical interface identity.
// memberEqPlan is the recursive equality plan of one interface member's
// exact dynamic type (value or pointer flavor). It returns a valid plan or
// an explicit error — never a fabricated fallback.
func (b *builder) memberEqPlan(named *types.Named, pointer, extern bool, span Span) (*EqPlan, error) {
	if pointer {
		// A pointer dynamic value compares by identity (the boxed instance).
		return &EqPlan{Kind: EqIdentity}, nil
	}
	if extern {
		if !types.Comparable(named) {
			// Go PANICS comparing an uncomparable dynamic type (an external
			// slice/map/func carrier, or a struct with such a field); emit
			// its exact runtime message rather than the unimplemented marker.
			return &EqPlan{Kind: EqUncomparable, Display: displayOf(named)}, nil
		}
		switch named.Underlying().(type) {
		case *types.Struct, *types.Array:
			// A comparable external struct/array compares field- or
			// element-wise in Go, but its TS value is an opaque handle whose
			// contents we do not own: the completion contract (a stub-provided
			// equality) is not yet declared, so equality fails closed loudly
			// at runtime rather than guessing === (wrong for distinct-but-equal
			// instances).
			return &EqPlan{Kind: EqExternal, Display: displayOf(named)}, nil
		}
		// A comparable basic/pointer/channel external carrier compares by ===.
		return &EqPlan{Kind: EqIdentity}, nil
	}
	return b.eqPlan(named, span)
}

// eqPlan builds the recursive typed equality plan of a value type. It
// descends ONLY into arrays: a struct compares through its own goEq$ and
// an interface through its own union equality, so the recursion never
// re-enters interface/struct resolution and is a finite tree. Construction
// is TOTAL: every admissible type form yields an explicit variant, a
// canonical-identity failure propagates unchanged (never masked as
// "uncomparable"), and any unhandled form fails closed with an error
// rather than a silent identity/=== default.
func (b *builder) eqPlan(t types.Type, span Span) (*EqPlan, error) {
	if _, isPtr := types.Unalias(t).Underlying().(*types.Pointer); isPtr {
		return &EqPlan{Kind: EqIdentity}, nil
	}
	if named, ok := types.Unalias(t).(*types.Named); ok {
		if pkg := named.Obj().Pkg(); pkg != nil && !b.unit.Owns(pkg.Path()) {
			if !types.Comparable(named) {
				return &EqPlan{Kind: EqUncomparable, Display: displayOf(t)}, nil
			}
			switch named.Underlying().(type) {
			case *types.Struct, *types.Array:
				return &EqPlan{Kind: EqExternal, Display: displayOf(t)}, nil
			case *types.Basic, *types.Pointer, *types.Chan:
				return &EqPlan{Kind: EqIdentity}, nil
			}
			return nil, &Unsupported{Kind: KindEqualityPlanForExternal, Code: "GOTOTS_UNSUPPORTED_TYPE",
				Construct: "equality plan for external " + t.String(), Span: span}
		}
	}
	switch u := types.Unalias(t).Underlying().(type) {
	case *types.Struct:
		if u.NumFields() == 0 {
			// The unit value: one value, one carrier (0) — identity.
			return &EqPlan{Kind: EqIdentity}, nil
		}
		if b.generatesGoEq(t) {
			return &EqPlan{Kind: EqGoEq}, nil
		}
		return &EqPlan{Kind: EqUncomparable, Display: displayOf(t)}, nil
	case *types.Array:
		if !types.Comparable(t) {
			return &EqPlan{Kind: EqUncomparable, Display: displayOf(t)}, nil
		}
		elem, err := b.eqPlan(u.Elem(), span)
		if err != nil {
			return nil, err
		}
		return &EqPlan{Kind: EqArray, Elem: elem}, nil
	case *types.Interface:
		// An interface array element compares through its own union
		// equality (the element's canonical interface identity). A
		// canonical-identity failure is a real construction error — it must
		// propagate, never collapse into "uncomparable".
		id, err := b.canonicalIfaceID(u)
		if err != nil {
			return nil, err
		}
		return &EqPlan{Kind: EqIface, IfaceID: id}, nil
	case *types.Slice, *types.Map, *types.Signature:
		// Go panics comparing these dynamic types inside an interface.
		return &EqPlan{Kind: EqUncomparable, Display: displayOf(t)}, nil
	case *types.Basic, *types.Chan:
		// A comparable basic / channel carrier compares by ===.
		return &EqPlan{Kind: EqIdentity}, nil
	}
	return nil, &Unsupported{Kind: KindEqualityPlanFor, Code: "GOTOTS_UNSUPPORTED_TYPE",
		Construct: "equality plan for " + t.String(), Span: span}
}

// generatesGoEq reports whether a struct type's generated class carries
// goEq$ — the SAME field admission as structEqComparable, but resolved
// through go/types directly so it never re-enters typeOf/ifaceMembers
// (that path recurses back here through interface fields). An owned
// concrete struct whose every field is exact-equality-comparable
// (scalars, pointers, unit, concrete interfaces, nested such structs, and
// arrays of those) gets goEq$; an external, slice, map, or function field
// keeps it out.
func (b *builder) generatesGoEq(t types.Type) bool {
	st, ok := types.Unalias(t).Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := range st.NumFields() {
		if !b.fieldEqComparable(st.Field(i).Type()) {
			return false
		}
	}
	return true
}

// fieldEqComparable is generatesGoEq's per-field predicate over go/types.
func (b *builder) fieldEqComparable(ft types.Type) bool {
	// An external named field (non-owned package) is excluded, exactly as
	// structEqComparable excludes KindExternal.
	if named, ok := types.Unalias(ft).(*types.Named); ok {
		if pkg := named.Obj().Pkg(); pkg != nil && !b.unit.Owns(pkg.Path()) {
			return false
		}
	}
	switch u := types.Unalias(ft).Underlying().(type) {
	case *types.Basic:
		return u.Info()&types.IsUntyped == 0
	case *types.Pointer:
		return true
	case *types.Interface:
		// A concrete interface field (its equality may panic, exactly Go);
		// a type-parameter constraint interface is instantiation-dependent
		// and stays out.
		_, isParam := types.Unalias(ft).(*types.TypeParam)
		return !isParam
	case *types.Array:
		return b.fieldEqComparable(u.Elem())
	case *types.Struct:
		return b.generatesGoEq(ft)
	}
	return false
}

// compositeEqPlan is a boxed composite's empty-interface equality plan.
func (b *builder) compositeEqPlan(source types.Type, span Span) (*EqPlan, error) {
	return b.eqPlan(source, span)
}

func (b *builder) ifaceMembers(iface *types.Interface, span Span) ([]IfaceMember, error) {
	key, err := b.canonicalIfaceID(iface)
	if err != nil {
		return nil, err
	}
	if cached, ok := b.unit.IfaceMemberCache(key); ok {
		return cached, nil
	}
	nested := b.ifaceMembersInProgress[key]
	if nested {
		// Re-entered while resolving an instance type: serve (and cache)
		// the NAMED members only; the outermost call computes and caches
		// the complete union.
		if cached, ok := b.unit.IfaceMemberCache("named-only:" + key); ok {
			return cached, nil
		}
	}
	if b.ifaceMembersInProgress == nil {
		b.ifaceMembersInProgress = map[string]bool{}
	}
	b.ifaceMembersInProgress[key] = true
	defer delete(b.ifaceMembersInProgress, key)
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
		if isStruct && !pointer {
			member.KeyEncodable = b.structKeyEncodable(named, span)
		}
		if member.Extern && !isStruct {
			// An external named type over a BASIC underlying boxes its
			// value carrier, not a branded handle.
			if basic, isBasic := named.Underlying().(*types.Basic); isBasic {
				member.ExternCarrier = basicCarrier(basic)
			}
		}
		if !member.Extern && !isStruct {
			if basic, isBasic := named.Underlying().(*types.Basic); isBasic {
				member.ValueCarrier = basicCarrier(basic)
			}
		}
		plan, err := b.memberEqPlan(named, pointer, member.Extern, span)
		if err != nil {
			return err
		}
		member.Eq = plan
		// Per-interface-method dispatch selectors: the member's own vtable
		// key for the method that implements each interface method. Bare in
		// the common case; disambiguated where the member promotes
		// same-bare-name methods from different packages (MethodSlot), so
		// dispatch and the member's vtable index the same canonical slot.
		set := types.NewMethodSet(named)
		if pointer {
			set = types.NewMethodSet(types.NewPointer(named))
		}
		slots := make(map[string]string, iface.NumMethods())
		for i := range iface.NumMethods() {
			m := iface.Method(i)
			sel := lookupSelection(set, m.Pkg(), m.Name())
			if sel == nil {
				// Membership was established by types.Implements, so the
				// member MUST resolve every interface method; a missing
				// selection is an invariant violation, not a slot to skip.
				return fmt.Errorf("ir: interface member %s implements %s but has no selection for method %s",
					named.Obj().Name(), key, m.Name())
			}
			impl, ok := sel.Obj().(*types.Func)
			if !ok {
				return fmt.Errorf("ir: interface member %s selection for method %s is not a func (%T)",
					named.Obj().Name(), m.Name(), sel.Obj())
			}
			slot, err := MethodSlot(named, impl)
			if err != nil {
				return err
			}
			// Keyed by the interface method's CANONICAL identity (not its
			// bare name), matching IfaceCall.MethodKey at the dispatch site.
			key, err := MethodKey(m)
			if err != nil {
				return err
			}
			slots[key] = slot
			if member.Extern {
				// An EXTERNAL member's vtable is built from its recorded
				// obligation: every interface method the membership proves
				// must be ON RECORD, or the vtable would lack the slot the
				// dispatch just resolved. Registration is idempotent and
				// validated by the sole constructor.
				if err := b.unit.AddExternalMethod(named, impl); err != nil {
					return err
				}
			}
		}
		member.Slots = slots
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
		// register records each interface method as a stub obligation for
		// this external implementer. It mirrors the fail-closed discipline of
		// the owned-member `add` path: membership was established by
		// types.Implements, so every interface method MUST resolve — a missing
		// selection is an invariant violation, a generic satisfying signature
		// is an unsupported external contract, and a canonical-identity
		// failure propagates. None is a silently dropped obligation.
		register := func(pointer bool) error {
			var t types.Type = named
			if pointer {
				t = types.NewPointer(named)
			}
			set := types.NewMethodSet(t)
			for i := range iface.NumMethods() {
				method := iface.Method(i)
				selection := lookupSelection(set, method.Pkg(), method.Name())
				if selection == nil {
					return fmt.Errorf("ir: external member %s implements %s but has no selection for method %s",
						named.Obj().Name(), key, method.Name())
				}
				fn := selection.Obj().(*types.Func)
				sig := fn.Type().(*types.Signature)
				if (sig.TypeParams() != nil && sig.TypeParams().Len() > 0) || SignatureMentionsTypeParam(sig) {
					// No single exact adapter over the stub export: fail closed
					// for the using body rather than dropping the obligation.
					return &Unsupported{Kind: KindGenericExternalMethodCall, Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
						Construct: "generic external method call", Span: span}
				}
				if err := b.unit.AddExternalMethod(named, fn); err != nil {
					return err
				}
			}
			return nil
		}
		if types.Implements(named, iface) {
			if err := add(named, false); err != nil {
				return nil, err
			}
			if err := register(false); err != nil {
				return nil, err
			}
		}
		if types.Implements(types.NewPointer(named), iface) {
			if err := add(named, true); err != nil {
				return nil, err
			}
			if err := register(true); err != nil {
				return nil, err
			}
		}
	}
	if nested {
		// Re-entered while resolving an instance type: the NAMED members
		// suffice for the nested spelling; the complete (cached) result
		// is computed by the outermost call.
		sort.Slice(members, func(i, j int) bool { return members[i].K < members[j].K })
		b.unit.SetIfaceMemberCache("named-only:"+key, members)
		return members, nil
	}
	// INSTANTIATED-GENERIC implementers: every concrete instantiation in
	// the closed evidence whose (pointer) method set satisfies the
	// interface joins the union as a composite-branded member with an
	// inline vtable over the generic functions. The candidate list and
	// each instantiation's slot surface are corpus-level caches — the
	// per-interface work is Implements plus member assembly only.
	for _, candidate := range b.instantiationCandidates() {
		if types.Implements(candidate.Named, iface) {
			member, err := b.instantiatedMember(candidate.Named, candidate.Canon, false, iface, span)
			if err != nil {
				return nil, err
			}
			if member != nil {
				members = append(members, *member)
			}
		}
		if types.Implements(types.NewPointer(candidate.Named), iface) {
			member, err := b.instantiatedMember(candidate.Named, candidate.Canon, true, iface, span)
			if err != nil {
				return nil, err
			}
			if member != nil {
				members = append(members, *member)
			}
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

// mentionsTypeParamType reports whether a Go type references any type
// parameter.
func mentionsTypeParamType(t types.Type) bool {
	sig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "", t)), types.NewTuple(), false)
	return SignatureMentionsTypeParam(sig)
}
