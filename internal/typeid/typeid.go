// Package typeid renders the ONE canonical semantic identity of a Go
// type, shared by the census shapes and the translator so their
// evidence joins byte-exactly. The identity is fully semantic:
//
//   - every package is qualified by IMPORT PATH (names collide);
//   - aliases are resolved recursively (an alias is not a distinct
//     dynamic type);
//   - unexported FIELD and METHOD names inside structural types carry
//     their declaring package (two structurally spelled-alike types
//     with unexported members from different packages are distinct).
package typeid

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// Canonical renders the canonical identity string. It is a TOTAL
// contract: an unhandled type form (no exact identity) returns an error
// rather than a poisoned string, so no caller can consume a
// non-canonical identity and the dual "string plus optional check"
// contract is eliminated.
func Canonical(t types.Type) (string, error) {
	var out strings.Builder
	write(&out, t, map[types.Type]bool{})
	s := out.String()
	if strings.Contains(s, unsupportedMarker) {
		return "", fmt.Errorf("typeid: no exact canonical identity for type form %T (%s)", t, t.String())
	}
	return s, nil
}

func write(out *strings.Builder, t types.Type, seen map[types.Type]bool) {
	t = types.Unalias(t)
	switch u := t.(type) {
	case *types.Basic:
		out.WriteString(u.Name())
	case *types.Named:
		obj := u.Obj()
		if obj.Pkg() != nil {
			out.WriteString(obj.Pkg().Path())
			out.WriteString(".")
		}
		out.WriteString(obj.Name())
		if args := u.TypeArgs(); args != nil && args.Len() > 0 {
			out.WriteString("[")
			for i := range args.Len() {
				if i > 0 {
					out.WriteString(",")
				}
				write(out, args.At(i), seen)
			}
			out.WriteString("]")
		}
	case *types.Pointer:
		out.WriteString("*")
		write(out, u.Elem(), seen)
	case *types.Slice:
		out.WriteString("[]")
		write(out, u.Elem(), seen)
	case *types.Array:
		fmt.Fprintf(out, "[%d]", u.Len())
		write(out, u.Elem(), seen)
	case *types.Map:
		out.WriteString("map[")
		write(out, u.Key(), seen)
		out.WriteString("]")
		write(out, u.Elem(), seen)
	case *types.Chan:
		switch u.Dir() {
		case types.SendOnly:
			out.WriteString("chan<- ")
		case types.RecvOnly:
			out.WriteString("<-chan ")
		default:
			out.WriteString("chan ")
		}
		write(out, u.Elem(), seen)
	case *types.Signature:
		if seen[t] {
			out.WriteString("func(...)")
			return
		}
		seen[t] = true
		// The RECEIVER is NOT part of a signature's identity: Go ignores it
		// when comparing signatures (go/types signature.go: "It is ignored
		// when comparing signatures for identity"), and an abstract
		// method's receiver is the ENCLOSING interface, so including it
		// would give structurally identical interfaces (interface{M()})
		// distinct identities. Receiver identity belongs in the
		// method-declaration shape (package + receiver + name + this
		// callable signature), not in the callable signature itself.
		//
		// Only the SIGNATURE's OWN type parameters are part of callable
		// identity (Go compares TypeParams(), params, results, variadic —
		// predicates.go), written by binder POSITION so alpha-equivalent
		// declarations ([T any] and [U any]) coincide. RecvTypeParams are
		// NOT part of the callable signature: they belong to the receiver
		// and are verified in the method-declaration shape. A generic
		// method references them in its params, resolved by binder index.
		writeTypeParams(out, u.TypeParams(), seen)
		out.WriteString("func(")
		params := u.Params()
		for i := range params.Len() {
			if i > 0 {
				out.WriteString(",")
			}
			if u.Variadic() && i == params.Len()-1 {
				out.WriteString("...")
			}
			write(out, params.At(i).Type(), seen)
		}
		out.WriteString(")")
		results := u.Results()
		if results.Len() > 0 {
			out.WriteString("(")
			for i := range results.Len() {
				if i > 0 {
					out.WriteString(",")
				}
				write(out, results.At(i).Type(), seen)
			}
			out.WriteString(")")
		}
		delete(seen, t)
	case *types.Struct:
		if seen[t] {
			out.WriteString("struct{...}")
			return
		}
		seen[t] = true
		out.WriteString("struct{")
		for i := range u.NumFields() {
			if i > 0 {
				out.WriteString(";")
			}
			field := u.Field(i)
			// An embedded field (struct{T}) promotes T's methods and fields
			// where a named field of the same spelling (struct{T T}) does
			// not: tag the anonymous case so the two never share identity.
			if field.Anonymous() {
				out.WriteString("embed ")
			}
			writeMemberName(out, field.Name(), field.Exported(), field.Pkg())
			out.WriteString(" ")
			write(out, field.Type(), seen)
			if tag := u.Tag(i); tag != "" {
				fmt.Fprintf(out, " %q", tag)
			}
		}
		out.WriteString("}")
		delete(seen, t)
	case *types.Interface:
		if seen[t] {
			out.WriteString("interface{...}")
			return
		}
		seen[t] = true
		// Interface identity is the NORMALIZED TYPE SET, not the syntax
		// (Go: predicates.go compares typeSet.terms, comparable, and the
		// full method set). So interface{Base} and interface{M()} coincide
		// when Base is interface{M()}, and interface{~int|~string} equals
		// interface{~string|~int}.
		out.WriteString("interface{")
		// The COMPLETE method set (NumMethods already includes embedded
		// methods); Go sorts methods by name, but sort defensively.
		methods := make([]string, 0, u.NumMethods())
		for i := range u.NumMethods() {
			method := u.Method(i)
			var m strings.Builder
			writeMemberName(&m, method.Name(), method.Exported(), method.Pkg())
			write(&m, method.Type(), seen)
			methods = append(methods, m.String())
		}
		sort.Strings(methods)
		for i, m := range methods {
			if i > 0 {
				out.WriteString(";")
			}
			out.WriteString(m)
		}
		// The type-set terms: embedded elements flattened to their terms
		// (embedded method interfaces contribute methods above, not terms),
		// unions expanded, each sorted and de-duplicated so syntactic order
		// and embedding shape never change identity. `comparable` is a term.
		terms := dedupeSorted(collectTerms(u, seen))
		for _, term := range terms {
			out.WriteString(";term:")
			out.WriteString(term)
		}
		out.WriteString("}")
		delete(seen, t)
	case *types.TypeParam:
		// A type parameter is identified by its BINDER POSITION, not its
		// source name, so alpha-equivalent declarations coincide.
		fmt.Fprintf(out, "$#%d", u.Index())
	case *types.Union:
		// A bare constraint union (int | ~string | …): normalized —
		// sorted and de-duplicated — so syntactic order does not matter
		// (union.go: terms are stored non-canonically).
		out.WriteString("union{")
		out.WriteString(strings.Join(dedupeSorted(unionTerms(u, seen)), "|"))
		out.WriteString("}")
	case *types.Tuple:
		for i := range u.Len() {
			if i > 0 {
				out.WriteString(",")
			}
			write(out, u.At(i).Type(), seen)
		}
	default:
		// Every well-formed type form is handled above. An unhandled form
		// is a construction gap, not a spelling to guess: emit a poison
		// marker that can never collide with a real identity and is
		// detectable (typeid.HasUnsupported), rather than falling back to
		// the ambiguous, non-package-qualified t.String() spelling.
		out.WriteString(unsupportedMarker)
		out.WriteString(t.String())
	}
}

// unsupportedMarker prefixes any identity that could not be built
// exactly; it contains a NUL so it can never appear in a real spelling.
const unsupportedMarker = "\x00!typeid-unsupported:"

// HasUnsupported reports whether a canonical identity was built over an
// unsupported type form and is therefore not an exact identity.
func HasUnsupported(id string) bool {
	return strings.Contains(id, unsupportedMarker)
}

// writeTypeParams writes a type-parameter list by binder POSITION with
// each parameter's complete constraint type set, so signatures that
// differ only in a constraint have distinct identities while
// alpha-equivalent renamings ([T any] vs [U any]) coincide.
func writeTypeParams(out *strings.Builder, params *types.TypeParamList, seen map[types.Type]bool) {
	if params == nil || params.Len() == 0 {
		return
	}
	out.WriteString("[")
	for i := range params.Len() {
		if i > 0 {
			out.WriteString(",")
		}
		fmt.Fprintf(out, "#%d:", i)
		write(out, params.At(i).Constraint(), seen)
	}
	out.WriteString("]")
}

// collectTerms flattens an interface's type-set restriction into its
// terms. Embedded METHOD interfaces contribute no terms (their methods
// are in the complete method set); embedded unions expand to their
// terms; an embedded concrete type or `comparable` is itself a term.
func collectTerms(iface *types.Interface, seen map[types.Type]bool) []string {
	var terms []string
	for i := range iface.NumEmbeddeds() {
		terms = append(terms, termsOf(iface.EmbeddedType(i), seen)...)
	}
	return terms
}

// termsOf renders the type-set terms contributed by one embedded element.
func termsOf(t types.Type, seen map[types.Type]bool) []string {
	switch e := t.(type) {
	case *types.Union:
		return unionTerms(e, seen)
	case *types.Interface:
		return collectTerms(e, seen)
	case *types.Named:
		// `comparable` (universe) is a type-set restriction, a term. An
		// embedded named INTERFACE contributes its terms (its methods are
		// already in the complete method set); any other named type is a
		// single-type restriction term.
		if e.Obj().Pkg() == nil && e.Obj().Name() == "comparable" {
			return []string{"comparable"}
		}
		if iface, ok := e.Underlying().(*types.Interface); ok {
			return collectTerms(iface, seen)
		}
		return []string{canonicalString(t, seen)}
	default:
		return []string{canonicalString(t, seen)}
	}
}

// unionTerms renders each union term with tilde approximation preserved.
func unionTerms(u *types.Union, seen map[types.Type]bool) []string {
	terms := make([]string, 0, u.Len())
	for i := range u.Len() {
		term := u.Term(i)
		prefix := ""
		if term.Tilde() {
			prefix = "~"
		}
		terms = append(terms, prefix+canonicalString(term.Type(), seen))
	}
	return terms
}

// canonicalString renders one type to its canonical identity string.
func canonicalString(t types.Type, seen map[types.Type]bool) string {
	var b strings.Builder
	write(&b, t, seen)
	return b.String()
}

// dedupeSorted sorts and removes duplicate terms so element order and
// repetition never affect identity.
func dedupeSorted(in []string) []string {
	if len(in) == 0 {
		return in
	}
	sort.Strings(in)
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}

// writeMemberName qualifies unexported member names with their declaring
// package: Go's identity rule for unexported fields and methods.
func writeMemberName(out *strings.Builder, name string, exported bool, pkg *types.Package) {
	if !exported && pkg != nil {
		out.WriteString(pkg.Path())
		out.WriteString("!")
	}
	out.WriteString(name)
}
