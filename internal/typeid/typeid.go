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

// Canonical renders the canonical identity string.
func Canonical(t types.Type) string {
	var out strings.Builder
	write(&out, t, map[types.Type]bool{})
	return out.String()
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
		// Type parameters and their CONSTRAINTS ARE part of identity
		// (func F[T any] and func F[T comparable] differ), written by
		// binder POSITION so alpha-equivalent declarations ([T any] and
		// [U any]) coincide. A method's parameters come from its receiver
		// (RecvTypeParams); a function's from TypeParams; the two are
		// mutually exclusive.
		writeTypeParams(out, u.RecvTypeParams(), seen)
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
		out.WriteString("interface{")
		for i := range u.NumMethods() {
			if i > 0 {
				out.WriteString(";")
			}
			method := u.Method(i)
			writeMemberName(out, method.Name(), method.Exported(), method.Pkg())
			write(out, method.Type(), seen)
		}
		// The COMPLETE type set: embedded elements (constraint unions,
		// ~terms, comparable, embedded interfaces) are part of identity, so
		// a methodless constraint interface{ ~int32 | ~uint32 } does not
		// collapse toward interface{}. Sorted by their canonical spelling
		// so element order does not affect identity.
		if n := u.NumEmbeddeds(); n > 0 {
			terms := make([]string, 0, n)
			for i := range n {
				var term strings.Builder
				write(&term, u.EmbeddedType(i), seen)
				terms = append(terms, term.String())
			}
			sort.Strings(terms)
			for _, term := range terms {
				out.WriteString(";elem:")
				out.WriteString(term)
			}
		}
		out.WriteString("}")
		delete(seen, t)
	case *types.TypeParam:
		// A type parameter is identified by its BINDER POSITION, not its
		// source name, so alpha-equivalent declarations coincide.
		fmt.Fprintf(out, "$#%d", u.Index())
	case *types.Union:
		// A constraint union (int | ~string | …): each term path-qualified
		// so terms from different packages never collide.
		out.WriteString("union{")
		for i := range u.Len() {
			if i > 0 {
				out.WriteString("|")
			}
			term := u.Term(i)
			if term.Tilde() {
				out.WriteString("~")
			}
			write(out, term.Type(), seen)
		}
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

// writeMemberName qualifies unexported member names with their declaring
// package: Go's identity rule for unexported fields and methods.
func writeMemberName(out *strings.Builder, name string, exported bool, pkg *types.Package) {
	if !exported && pkg != nil {
		out.WriteString(pkg.Path())
		out.WriteString("!")
	}
	out.WriteString(name)
}
