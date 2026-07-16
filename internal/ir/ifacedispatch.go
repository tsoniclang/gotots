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
)

// resolveIfaceBranches computes the closed dispatch set for one
// interface method call: every concrete type in the unit that
// implements the receiver's interface, with the direct method to call.
func (b *builder) resolveIfaceBranches(ifaceType *types.Interface, method *types.Func, span Span) ([]IfaceBranch, error) {
	target := method.Name()
	var branches []IfaceBranch
	seen := map[string]bool{}
	add := func(named *types.Named, pointer bool) error {
		var t types.Type = named
		if pointer {
			t = types.NewPointer(named)
		}
		set := types.NewMethodSet(t)
		selection := lookupSelection(set, named.Obj().Pkg(), target)
		if selection == nil {
			return nil
		}
		obj := named.Obj()
		key := obj.Pkg().Path() + "." + obj.Name()
		if pointer {
			key = "*" + key
		}
		if seen[key] {
			return nil
		}
		seen[key] = true
		payload, err := b.typeOf(t, span)
		if err != nil {
			return err
		}
		rtti, err := b.rttiFor(t, span)
		if err != nil {
			return err
		}
		method := selection.Obj().(*types.Func)
		methodRecv := method.Type().(*types.Signature).Recv()
		declNamed, ok := types.Unalias(methodRecv.Type()).(*types.Named)
		if pointerType, isPtr := methodRecv.Type().(*types.Pointer); isPtr {
			declNamed, ok = types.Unalias(pointerType.Elem()).(*types.Named)
		}
		if !ok {
			return &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface dispatch to a method on an unnamed receiver", Span: span}
		}
		_, isPointerRecv := methodRecv.Type().(*types.Pointer)
		branch := IfaceBranch{
			Rtti:          rtti,
			Payload:       payload,
			DeclPkg:       declNamed.Obj().Pkg().Path(),
			DeclType:      declNamed.Obj().Name(),
			External:      !b.unit.Owns(declNamed.Obj().Pkg().Path()),
			Method:        method.Name(),
			ValueReceiver: !isPointerRecv,
		}
		// A promoted method chains through the embedded value fields of
		// the concrete type; the selection index names that path.
		if path := selection.Index(); len(path) > 1 {
			current := named.Underlying()
			for _, index := range path[:len(path)-1] {
				structType, isStruct := current.(*types.Struct)
				if !isStruct {
					return &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
						Construct: "interface dispatch through non-struct promotion", Span: span}
				}
				field := structType.Field(index)
				step := PromotionStep{Field: field.Name()}
				resolved, err := b.typeOf(field.Type(), span)
				if err != nil {
					return err
				}
				step.FieldType = resolved
				fieldType := field.Type()
				if pointer, isPtr := fieldType.Underlying().(*types.Pointer); isPtr {
					step.Pointer = true
					fieldType = pointer.Elem()
				}
				branch.FieldPath = append(branch.FieldPath, step)
				current = fieldType.Underlying()
			}
		}
		branches = append(branches, branch)
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
			continue // generic types dispatch per instantiation, handled elsewhere
		}
		// A boxed value may be the value type or its pointer; both tokens
		// join the switch when they implement the interface. Whole-unit
		// coverage is sound: an over-approximated branch is dead, never a
		// missing case (which would be an unsound runtime panic).
		if types.Implements(named, ifaceType) {
			if err := add(named, false); err != nil {
				return nil, err
			}
		}
		if types.Implements(types.NewPointer(named), ifaceType) {
			if err := add(named, true); err != nil {
				return nil, err
			}
		}
	}
	return branches, nil
}

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
