// Instantiated-generic union members: concrete instantiations from the
// closed evidence join interface unions as composite-branded members
// with inline vtable surfaces over the generated generic functions.
package ir

import (
	"fmt"
	"go/types"
)

// instantiatedMember builds one instantiated-generic union member: the
// composite-branded payload plus the inline vtable surface (each slot
// dispatching to the generated generic function with the instantiation's
// exact types). A member whose surface is not representable (a slot type
// outside the reviewed subset) is SKIPPED — boxing it still fails closed
// at the box site.
func (b *builder) instantiatedMember(instNamed *types.Named, canon string, pointer bool, iface *types.Interface, span Span) (*IfaceMember, error) {
	instIR, err := b.typeOf(instNamed, span)
	if err != nil {
		return nil, nil
	}
	origin := instNamed.Origin()
	k := canon
	if pointer {
		k = "*" + canon
	}
	member := IfaceMember{
		K:   "c:" + k,
		Pkg: origin.Obj().Pkg().Path(), Type: origin.Obj().Name(), Pointer: pointer,
		Struct:    structUnderlying(instNamed),
		Composite: k,
		InstType:  instIR,
	}
	if member.Struct && !pointer {
		member.KeyEncodable = b.structKeyEncodable(instNamed, span)
	}
	plan, err := b.memberEqPlan(instNamed, pointer, false, span)
	if err != nil {
		return nil, nil
	}
	member.Eq = plan
	instSlots, ok, err := b.instantiationSlots(instNamed, pointer, span)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	member.InstSlots = instSlots
	bySlotName := map[string]bool{}
	for _, slot := range instSlots {
		bySlotName[slot.Slot] = true
	}
	var set *types.MethodSet
	if pointer {
		set = types.NewMethodSet(types.NewPointer(instNamed))
	} else {
		set = types.NewMethodSet(instNamed)
	}
	slots := make(map[string]string, iface.NumMethods())
	for i := range iface.NumMethods() {
		m := iface.Method(i)
		sel := lookupSelection(set, m.Pkg(), m.Name())
		if sel == nil {
			return nil, fmt.Errorf("ir: instantiated member %s implements the interface but has no selection for method %s", canon, m.Name())
		}
		impl, isFunc := sel.Obj().(*types.Func)
		if !isFunc {
			return nil, nil
		}
		slot, err := MethodSlot(instNamed, impl)
		if err != nil {
			return nil, nil
		}
		if !bySlotName[slot] {
			// A promoted or otherwise unrepresented slot: skip the member.
			return nil, nil
		}
		methodKey, err := MethodKey(m)
		if err != nil {
			return nil, err
		}
		slots[methodKey] = slot
	}
	member.Slots = slots
	return &member, nil
}

// instantiationCandidates returns the corpus-wide concrete-instantiation
// candidates (computed once per unit).
func (b *builder) instantiationCandidates() []InstCandidate {
	cached := b.unit.CachedInstCandidates()
	if cached != nil {
		return *cached
	}
	out := []InstCandidate{}
	for _, typeName := range b.unit.GenericTypeObjects() {
		if !b.unit.Owns(typeName.Pkg().Path()) {
			continue
		}
		origin, isNamed := typeName.Type().(*types.Named)
		if !isNamed {
			continue
		}
		if _, isIface := origin.Underlying().(*types.Interface); isIface {
			continue
		}
		if origin.Obj().Pkg() != nil && types.NewMethodSet(types.NewPointer(origin)).Len() == 0 {
			// No methods at all: never an implementer of a non-empty
			// interface, and the empty interface takes instances through
			// the boxed-composite log — skip the candidate outright.
			continue
		}
		seenVectors := map[string]bool{}
		for _, vector := range b.unit.GenericTypeInstances(typeName) {
			concrete := true
			for _, arg := range vector {
				if mentionsTypeParamType(arg) {
					concrete = false
					break
				}
			}
			if !concrete || len(vector) == 0 {
				continue
			}
			instType, err := types.Instantiate(nil, origin, vector, false)
			if err != nil {
				continue
			}
			instNamed, isNamed := instType.(*types.Named)
			if !isNamed {
				continue
			}
			canon, err := b.canonicalTypeID(instNamed)
			if err != nil || seenVectors[canon] {
				continue
			}
			seenVectors[canon] = true
			out = append(out, InstCandidate{Named: instNamed, Canon: canon})
		}
	}
	b.unit.SetCachedInstCandidates(out)
	return out
}

// instantiationSlots builds the FULL method-set vtable surface of one
// concrete generic instantiation: each slot dispatches to the generated
// generic function with the instantiation's exact types. ok=false when
// any part of the surface is outside the reviewed subset (the caller
// skips the member; boxing still fails closed at its own site).
func (b *builder) instantiationSlots(instNamed *types.Named, pointer bool, span Span) ([]InstSlot, bool, error) {
	cacheKey, cacheErr := b.canonicalTypeID(instNamed)
	if cacheErr == nil {
		if pointer {
			cacheKey = "*" + cacheKey
		}
		if entry, hit := b.unit.CachedInstSlots(cacheKey); hit {
			return entry.slots, entry.ok, nil
		}
		defer func() {}()
	}
	slots, ok, err := b.instantiationSlotsUncached(instNamed, pointer, span)
	if cacheErr == nil && err == nil {
		b.unit.SetCachedInstSlots(cacheKey, instSlotsEntry{slots: slots, ok: ok})
	}
	return slots, ok, err
}

func (b *builder) instantiationSlotsUncached(instNamed *types.Named, pointer bool, span Span) ([]InstSlot, bool, error) {
	origin := instNamed.Origin()
	keyedParams := make([]bool, 0, origin.TypeParams().Len())
	for i := range origin.TypeParams().Len() {
		keyedParams = append(keyedParams, b.unit.ParamRequiresKeyOp(origin.Obj(), i))
	}
	typeArgs := make([]Type, 0, instNamed.TypeArgs().Len())
	for i := range instNamed.TypeArgs().Len() {
		argIR, err := b.typeOf(instNamed.TypeArgs().At(i), span)
		if err != nil {
			return nil, false, nil
		}
		typeArgs = append(typeArgs, argIR)
	}
	var set *types.MethodSet
	if pointer {
		set = types.NewMethodSet(types.NewPointer(instNamed))
	} else {
		set = types.NewMethodSet(instNamed)
	}
	out := make([]InstSlot, 0, set.Len())
	for i := range set.Len() {
		sel := set.At(i)
		impl, isFunc := sel.Obj().(*types.Func)
		if !isFunc {
			return nil, false, nil
		}
		if len(sel.Index()) > 1 {
			// A promoted method: the instantiation's own surface only —
			// skip the member rather than mis-dispatching.
			return nil, false, nil
		}
		slot, err := MethodSlot(instNamed, impl)
		if err != nil {
			return nil, false, nil
		}
		signature, isSig := impl.Type().(*types.Signature)
		if !isSig {
			return nil, false, nil
		}
		instSlot := InstSlot{
			Slot:        slot,
			MethodName:  impl.Name(),
			TypeArgs:    typeArgs,
			KeyedParams: keyedParams,
		}
		if recv := signature.Recv(); recv != nil {
			_, instSlot.PointerRecv = recv.Type().(*types.Pointer)
		}
		for j := range signature.Params().Len() {
			paramIR, err := b.typeOf(signature.Params().At(j).Type(), span)
			if err != nil {
				return nil, false, nil
			}
			instSlot.Params = append(instSlot.Params, paramIR)
		}
		for j := range signature.Results().Len() {
			resultIR, err := b.typeOf(signature.Results().At(j).Type(), span)
			if err != nil {
				return nil, false, nil
			}
			instSlot.Results = append(instSlot.Results, resultIR)
		}
		out = append(out, instSlot)
	}
	return out, true, nil
}

// structUnderlying reports whether a named type's underlying is a struct.
func structUnderlying(named *types.Named) bool {
	_, isStruct := named.Underlying().(*types.Struct)
	return isStruct
}
