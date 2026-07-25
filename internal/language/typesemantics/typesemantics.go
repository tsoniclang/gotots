// Package typesemantics is the general exact Go type-semantics owner: normalized
// type-set and core-type behavior for constraints and type parameters. It is
// the single authority — construct-local flatten/sort/deduplicate
// approximations are forbidden and mutation-tested against its differential
// matrix.
package typesemantics

import "go/types"

// SetKind distinguishes the universal, finite, and empty normalized
// structural type sets. An empty slice alone cannot distinguish universe from
// empty and is therefore never the authority.
type SetKind uint8

const (
	SetInvalid SetKind = iota
	SetUniverse
	SetFinite
	SetEmpty
)

// Term is one exported normalized structural type-set term.
type Term struct {
	Type  types.Type
	Tilde bool
}

// NormalizedTerms computes the exact structural type set of an interface or
// type-parameter constraint. Methods and comparability remain separate facts.
func NormalizedTerms(
	typ types.Type,
) (SetKind, []Term, bool) {
	if parameter, ok := types.Unalias(typ).(*types.TypeParam); ok {
		typ = parameter.Constraint()
	}
	iface, ok := types.Unalias(typ).Underlying().(*types.Interface)
	if !ok {
		return SetInvalid, nil, false
	}
	set, ok := interfaceTermSet(iface)
	if !ok {
		return SetInvalid, nil, false
	}
	if set.universe {
		return SetUniverse, nil, true
	}
	if len(set.terms) == 0 {
		return SetEmpty, nil, true
	}
	out := make([]Term, 0, len(set.terms))
	for _, term := range set.terms {
		out = append(out, Term{
			Type: term.typ, Tilde: term.tilde,
		})
	}
	return SetFinite, out, true
}

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

// term is one normalized structural type-set term: a specific type, or a
// tilde term admitting every type with the given underlying.
type term struct {
	typ   types.Type
	tilde bool
}

// termSet is a normalized structural type set: nil terms with universe=true
// is the universe (an interface without structural terms); universe=false
// with no terms is the empty set.
type termSet struct {
	universe bool
	terms    []term
}

// constraintCore resolves a type parameter's constraint through exact
// normalized union/intersection algebra and computes the core type of the
// resulting term set.
func constraintCore(tp *types.TypeParam) (types.Type, bool) {
	constraint := tp.Constraint()
	if constraint == nil {
		return nil, false
	}
	iface, ok := types.Unalias(constraint).Underlying().(*types.Interface)
	if !ok {
		return nil, false
	}
	set, ok := interfaceTermSet(iface)
	if !ok || set.universe || len(set.terms) == 0 {
		return nil, false
	}
	return coreOfTerms(set.terms)
}

// interfaceTermSet computes an interface's structural term set: the
// INTERSECTION of its embedded elements' sets (methods restrict membership
// but never the structural terms).
func interfaceTermSet(iface *types.Interface) (termSet, bool) {
	result := termSet{universe: true}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		element, ok := elementTermSet(iface.EmbeddedType(i))
		if !ok {
			return termSet{}, false
		}
		result = intersect(result, element)
	}
	return result, true
}

// elementTermSet computes one embedded element's term set: a union is the
// UNION of its terms; an embedded interface is its own intersected set; a
// specific type is a single-term set.
func elementTermSet(embedded types.Type) (termSet, bool) {
	switch e := types.Unalias(embedded).(type) {
	case *types.Union:
		set := termSet{}
		for j := 0; j < e.Len(); j++ {
			t := e.Term(j)
			set.terms = appendTermDedup(
				set.terms,
				term{
					typ:   types.Unalias(t.Type()),
					tilde: t.Tilde(),
				},
			)
		}
		return set, true
	case *types.Interface:
		return interfaceTermSet(e)
	default:
		if nested, isIface := types.Unalias(embedded).Underlying().(*types.Interface); isIface {
			return interfaceTermSet(nested)
		}
		return termSet{terms: []term{{typ: types.Unalias(embedded)}}}, true
	}
}

// intersect computes the exact intersection of two term sets per the Go
// specification's term-intersection rules.
func intersect(a, b termSet) termSet {
	if a.universe {
		return b
	}
	if b.universe {
		return a
	}
	out := termSet{}
	for _, left := range a.terms {
		for _, right := range b.terms {
			if merged, ok := intersectTerms(left, right); ok {
				out.terms = appendTermDedup(out.terms, merged)
			}
		}
	}
	return out
}

// intersectTerms intersects two terms: specific∩specific needs identity;
// specific∩tilde needs matching underlying (result specific); tilde∩tilde
// needs identical underlying (result tilde).
func intersectTerms(a, b term) (term, bool) {
	switch {
	case !a.tilde && !b.tilde:
		if types.Identical(a.typ, b.typ) {
			return a, true
		}
	case a.tilde && b.tilde:
		if types.Identical(a.typ.Underlying(), b.typ.Underlying()) {
			return a, true
		}
	case a.tilde:
		if types.Identical(b.typ.Underlying(), a.typ.Underlying()) {
			return b, true
		}
	default:
		if types.Identical(a.typ.Underlying(), b.typ.Underlying()) {
			return a, true
		}
	}
	return term{}, false
}

func appendTermDedup(terms []term, t term) []term {
	for _, existing := range terms {
		if existing.tilde == t.tilde && types.Identical(existing.typ, t.typ) {
			return terms
		}
	}
	return append(terms, t)
}

// coreOfTerms computes the core type of a normalized non-empty term set:
// one common underlying type, or the channel merge with the most restrictive
// shared direction.
func coreOfTerms(terms []term) (types.Type, bool) {
	var common types.Type
	commonOK := true
	for _, t := range terms {
		u := t.typ.Underlying()
		if common == nil {
			common = u
		} else if !types.Identical(common, u) {
			commonOK = false
		}
	}
	if commonOK {
		return common, true
	}
	var elem types.Type
	direction := types.SendRecv
	for _, t := range terms {
		channel, ok := t.typ.Underlying().(*types.Chan)
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
				return nil, false
			}
			direction = channel.Dir()
		}
	}
	return types.NewChan(direction, elem), true
}
