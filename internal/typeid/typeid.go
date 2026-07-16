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
	case *types.Tuple:
		for i := range u.Len() {
			if i > 0 {
				out.WriteString(",")
			}
			write(out, u.At(i).Type(), seen)
		}
	default:
		out.WriteString(t.String())
	}
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
