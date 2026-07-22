// Package typeset is the general exact Go type-semantics owner: normalized
// type-set and core-type behavior for constraints and type parameters. It is
// the single authority — construct-local flatten/sort/deduplicate
// approximations are forbidden and mutation-tested against its differential
// matrix.
package typeset

import "go/types"

// Core computes the core type of t per the Go specification's core-type
// definition, resolving type parameters through their constraint's type set:
//
//   - a type set with one common underlying type U has core type U;
//   - a type set whose members are all channel types with identical element
//     type E, containing both directional and bidirectional channels only if
//     every directional member agrees on direction, has core type chan E with
//     the most restrictive direction present;
//   - an empty or mixed type set has no core type (ok=false).
//
// Non-type-parameter types resolve to their alias-free underlying type.
func Core(t types.Type) (types.Type, bool) {
	if tp, ok := types.Unalias(t).(*types.TypeParam); ok {
		return constraintCore(tp)
	}
	u := types.Unalias(t).Underlying()
	if u == nil {
		return nil, false
	}
	return u, true
}

// constraintCore resolves a type parameter's constraint type set.
func constraintCore(tp *types.TypeParam) (types.Type, bool) {
	constraint := tp.Constraint()
	if constraint == nil {
		return nil, false
	}
	iface, ok := types.Unalias(constraint).Underlying().(*types.Interface)
	if !ok {
		return nil, false
	}
	terms, ok := typeSetTerms(iface)
	if !ok || len(terms) == 0 {
		// An empty term list means an unconstrained (any-like) interface —
		// no structural terms, hence no core type.
		return nil, false
	}
	// Single common underlying type across every term.
	var common types.Type
	commonOK := true
	for _, term := range terms {
		u := types.Unalias(term).Underlying()
		if common == nil {
			common = u
		} else if !types.Identical(common, u) {
			commonOK = false
		}
	}
	if commonOK {
		return common, true
	}
	// Channel merge: identical element type, compatible directions.
	var elem types.Type
	direction := types.SendRecv
	for _, term := range terms {
		channel, ok := types.Unalias(term).Underlying().(*types.Chan)
		if !ok {
			return nil, false
		}
		if elem == nil {
			elem = channel.Elem()
		} else if !types.Identical(elem, channel.Elem()) {
			return nil, false
		}
		if channel.Dir() != types.SendRecv {
			if direction != types.SendRecv && direction != channel.Dir() {
				return nil, false // conflicting directional members
			}
			direction = channel.Dir()
		}
	}
	return types.NewChan(direction, elem), true
}

// typeSetTerms flattens an interface's structural type-set terms: unions,
// embedded interfaces (intersection), and tilde terms (which contribute their
// underlying type to core-type computation). A term list is invalid (ok=false)
// when an embedded form is not resolvable structurally.
func typeSetTerms(iface *types.Interface) ([]types.Type, bool) {
	var terms []types.Type
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		embedded := iface.EmbeddedType(i)
		switch e := types.Unalias(embedded).(type) {
		case *types.Union:
			for j := 0; j < e.Len(); j++ {
				term := e.Term(j)
				if term.Tilde() {
					// ~T contributes underlying(T) to the core computation.
					terms = append(terms, types.Unalias(term.Type()).Underlying())
				} else {
					terms = append(terms, term.Type())
				}
			}
		case *types.Interface:
			// Intersection: embedded interface terms constrain further. Core
			// type of an intersection is computed over the combined terms; an
			// embedded empty interface adds nothing.
			nested, ok := typeSetTerms(e)
			if !ok {
				return nil, false
			}
			terms = append(terms, nested...)
		default:
			// A named/specific embedded type is a single-term constraint.
			if _, isIface := types.Unalias(embedded).Underlying().(*types.Interface); isIface {
				nested, ok := typeSetTerms(types.Unalias(embedded).Underlying().(*types.Interface))
				if !ok {
					return nil, false
				}
				terms = append(terms, nested...)
				continue
			}
			terms = append(terms, embedded)
		}
	}
	return terms, true
}
