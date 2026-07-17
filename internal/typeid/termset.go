// Constraint type-set algebra, ported faithfully from go/types
// (typeterm.go / termlist.go). An interface's type set is the
// INTERSECTION of its embedded elements' type sets, not a concatenation
// of syntactic terms — so interface{int|string} ({int,string}) and
// interface{int; string} (∅) are correctly distinct, while
// interface{int|string; int} and interface{int} both reduce to {int}.
// term equality/intersection use types.Identical, exactly as Go does, so
// the resulting identity agrees with types.Identical.
package typeid

import (
	"go/types"
	"sort"
	"strings"
)

// term describes an elementary type set: nil ⇒ ∅ (no types); {typ:nil} ⇒
// 𝓤 (all types); {false,T} ⇒ {T}; {true,t} ⇒ {t' | under(t')==t}.
type term struct {
	tilde bool
	typ   types.Type
}

func (x *term) disjoint(y *term) bool {
	ux := x.typ
	if y.tilde {
		ux = ux.Underlying()
	}
	uy := y.typ
	if x.tilde {
		uy = uy.Underlying()
	}
	return !types.Identical(ux, uy)
}

func (x *term) union(y *term) (*term, *term) {
	switch {
	case x == nil && y == nil:
		return nil, nil
	case x == nil:
		return y, nil
	case y == nil:
		return x, nil
	case x.typ == nil:
		return x, nil
	case y.typ == nil:
		return y, nil
	}
	if x.disjoint(y) {
		return x, y
	}
	if x.tilde || !y.tilde {
		return x, nil
	}
	return y, nil
}

func (x *term) intersect(y *term) *term {
	switch {
	case x == nil || y == nil:
		return nil
	case x.typ == nil:
		return y
	case y.typ == nil:
		return x
	}
	if x.disjoint(y) {
		return nil
	}
	if !x.tilde || y.tilde {
		return x
	}
	return y
}

// termlist is the union of its terms; ∅ is the empty list, 𝓤 is a list
// containing the {typ:nil} term.
type termlist []*term

func allTermlist() termlist { return termlist{new(term)} }

func (xl termlist) isEmpty() bool {
	for _, x := range xl {
		if x != nil {
			return false
		}
	}
	return true
}

func (xl termlist) isAll() bool {
	for _, x := range xl {
		if x != nil && x.typ == nil {
			return true
		}
	}
	return false
}

func (xl termlist) norm() termlist {
	used := make([]bool, len(xl))
	var rl termlist
	for i, xi := range xl {
		if xi == nil || used[i] {
			continue
		}
		for j := i + 1; j < len(xl); j++ {
			xj := xl[j]
			if xj == nil || used[j] {
				continue
			}
			if u1, u2 := xi.union(xj); u2 == nil {
				if u1.typ == nil {
					return allTermlist()
				}
				xi = u1
				used[j] = true
			}
		}
		rl = append(rl, xi)
	}
	return rl
}

func (xl termlist) intersect(yl termlist) termlist {
	if xl.isEmpty() || yl.isEmpty() {
		return nil
	}
	var rl termlist
	for _, x := range xl {
		for _, y := range yl {
			if r := x.intersect(y); r != nil {
				rl = append(rl, r)
			}
		}
	}
	return rl.norm()
}

// interfaceTerms computes the normalized type-set terms of an interface
// and whether it embeds `comparable`. It returns an error if a term type
// has no exact canonical identity.
func interfaceTerms(iface *types.Interface, c *idctx) (termlist, bool, error) {
	terms := allTermlist()
	comparable := false
	for i := range iface.NumEmbeddeds() {
		sub, subComparable, err := embeddedTerms(iface.EmbeddedType(i), c)
		if err != nil {
			return nil, false, err
		}
		comparable = comparable || subComparable
		terms = terms.intersect(sub)
	}
	// `comparable` filters finite terms to comparable ones; over 𝓤 it is a
	// genuine restriction carried in serialization.
	if comparable && !terms.isAll() && !terms.isEmpty() {
		var kept termlist
		for _, t := range terms {
			if t != nil && t.typ != nil && types.Comparable(t.typ) {
				kept = append(kept, t)
			}
		}
		terms = kept
	}
	return terms, comparable, nil
}

// embeddedTerms renders one embedded element's type set.
func embeddedTerms(t types.Type, c *idctx) (termlist, bool, error) {
	switch e := t.(type) {
	case *types.Union:
		var tl termlist
		for i := range e.Len() {
			ut := e.Term(i)
			tl = append(tl, &term{tilde: ut.Tilde(), typ: ut.Type()})
		}
		return tl.norm(), false, nil
	case *types.Interface:
		return interfaceTerms(e, c)
	case *types.Named:
		if e.Obj().Pkg() == nil && e.Obj().Name() == "comparable" {
			return allTermlist(), true, nil
		}
		if iface, ok := e.Underlying().(*types.Interface); ok {
			return interfaceTerms(iface, c)
		}
		return termlist{&term{typ: e}}, false, nil
	default:
		return termlist{&term{typ: t}}, false, nil
	}
}

// serializeTermset renders a normalized type set canonically: methods are
// written by the caller; this returns the term-restriction suffix.
func serializeTermset(terms termlist, comparable bool, c *idctx) (string, error) {
	switch {
	case terms.isEmpty():
		return ";termset:∅", nil
	case terms.isAll():
		if comparable {
			return ";termset:comparable", nil
		}
		return "", nil // no type restriction (a plain method interface)
	}
	parts := make([]string, 0, len(terms))
	for _, t := range terms {
		if t == nil || t.typ == nil {
			continue
		}
		// Write within the CURRENT binder context: a term like ~[]E
		// references a type parameter bound by an enclosing generic
		// signature, which a fresh Canonical() would not have.
		child := c.child()
		var b strings.Builder
		write(&b, t.typ, child)
		if *child.errp != nil {
			return "", *child.errp
		}
		s := b.String()
		if t.tilde {
			s = "~" + s
		}
		parts = append(parts, s)
	}
	sort.Strings(parts)
	return ";termset:" + strings.Join(parts, "|"), nil
}
