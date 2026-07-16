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
		// The receiver is part of a method's identity: func (T) M and
		// func (*T) M are DIFFERENT methods with different method sets, and
		// must never share an identity.
		if recv := u.Recv(); recv != nil {
			out.WriteString("recv(")
			write(out, recv.Type(), seen)
			out.WriteString(")")
		}
		// Type parameters and their CONSTRAINTS are part of a generic
		// signature's identity: func F[T any] and func F[T comparable]
		// differ in what T admits and must not collide.
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
		out.WriteString("}")
		delete(seen, t)
	case *types.TypeParam:
		out.WriteString("$")
		out.WriteString(u.Obj().Name())
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

// writeTypeParams writes a type-parameter list with each parameter's
// constraint, so two generic signatures that differ only in a constraint
// have distinct identities.
func writeTypeParams(out *strings.Builder, params *types.TypeParamList, seen map[types.Type]bool) {
	if params == nil || params.Len() == 0 {
		return
	}
	out.WriteString("[")
	for i := range params.Len() {
		if i > 0 {
			out.WriteString(",")
		}
		param := params.At(i)
		out.WriteString(param.Obj().Name())
		out.WriteString(":")
		write(out, param.Constraint(), seen)
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
