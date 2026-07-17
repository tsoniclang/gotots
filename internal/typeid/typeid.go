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
	c := newCtx()
	write(&out, t, c)
	if *c.errp != nil {
		return "", *c.errp
	}
	return out.String(), nil
}

// MethodCanonical builds a method's callable identity WITH its receiver's
// type parameters bound: a free receiver-bound parameter is never a
// global index. The receiver constraints are written before the callable
// signature so two methods that differ only in a receiver constraint
// (A[T any].M vs B[U ~int].M) have distinct identities.
func MethodCanonical(method *types.Func) (string, error) {
	sig, ok := method.Type().(*types.Signature)
	if !ok {
		return "", fmt.Errorf("typeid: method %s has no signature", method.Name())
	}
	c := newCtx()
	if rtp := sig.RecvTypeParams(); rtp != nil && rtp.Len() > 0 {
		// A free receiver parameter is distinct PER OWNER even with an
		// identical constraint (go/types: A[T any].F()T is not identical to
		// B[U any].F()U), so it is bound to its owner's identity — never a
		// bare index or its constraint (a method with no free parameter
		// appearing, A.G() vs B.G(), stays identical). If the owner cannot
		// be proven, identity fails closed.
		owner := receiverOwnerID(sig)
		if owner == "" {
			return "", fmt.Errorf("typeid: method %s has receiver type parameters but no provable owner", method.Name())
		}
		for i := range rtp.Len() {
			c.bindOwner(rtp.At(i), owner, i)
		}
	}
	var out strings.Builder
	writeSignatureBody(&out, sig, c)
	if *c.errp != nil {
		return "", *c.errp
	}
	return out.String(), nil
}

// receiverOwnerID is the canonical identity of the generic type that owns
// a method's receiver type parameters.
func receiverOwnerID(sig *types.Signature) string {
	recv := sig.Recv()
	if recv == nil {
		return ""
	}
	t := recv.Type()
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	obj := named.Obj()
	if obj.Pkg() == nil {
		return obj.Name()
	}
	return obj.Pkg().Path() + "." + obj.Name()
}

// Binders is an explicit type-parameter binder environment for
// canonicalizing a type that references the parameters of an enclosing
// generic declaration (a field or parameter type inside a generic
// struct/function). A free parameter outside this environment still
// fails closed.
type Binders struct{ c *idctx }

// OwnerBinders binds the type parameters of a generic named type, so a
// component type like []T inside Container[T] canonicalizes with T
// resolved to its owner. Returns nil-safe empty binders for non-generic
// owners.
func OwnerBinders(owner *types.Named) *Binders {
	c := newCtx()
	if owner != nil {
		if tp := owner.TypeParams(); tp != nil && tp.Len() > 0 {
			id := owner.Obj().Name()
			if owner.Obj().Pkg() != nil {
				id = owner.Obj().Pkg().Path() + "." + owner.Obj().Name()
			}
			for i := range tp.Len() {
				c.bindOwner(tp.At(i), id, i)
			}
		}
	}
	return &Binders{c: c}
}

// FuncBinders binds a signature's OWN type parameters (a generic
// function), so its parameter/result component types canonicalize with
// those parameters resolved by binder position.
func FuncBinders(sig *types.Signature) *Binders {
	c := newCtx()
	if sig != nil {
		if tp := sig.TypeParams(); tp != nil && tp.Len() > 0 {
			for i := range tp.Len() {
				c.bind(tp.At(i), i)
			}
		}
	}
	return &Binders{c: c}
}

// MethodBinders binds a method's receiver type parameters, for
// canonicalizing its parameter/result component types.
func MethodBinders(fn *types.Func) (*Binders, error) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("typeid: %s has no signature", fn.Name())
	}
	c := newCtx()
	if rtp := sig.RecvTypeParams(); rtp != nil && rtp.Len() > 0 {
		owner := receiverOwnerID(sig)
		if owner == "" {
			return nil, fmt.Errorf("typeid: method %s has receiver type parameters but no provable owner", fn.Name())
		}
		for i := range rtp.Len() {
			c.bindOwner(rtp.At(i), owner, i)
		}
	}
	return &Binders{c: c}, nil
}

// Canonical renders one type's identity within this binder environment.
// A Binders is reusable across many types, so each call gets a FRESH
// error slot (and cycle guard) — one failure never poisons the next.
func (b *Binders) Canonical(t types.Type) (string, error) {
	c := &idctx{seen: map[types.Type]bool{}, binders: b.c.binders, errp: new(error)}
	var out strings.Builder
	write(&out, t, c)
	if *c.errp != nil {
		return "", *c.errp
	}
	return out.String(), nil
}

// DeclSignature is the canonical signature identity of a declared
// function or method: a method routes through MethodCanonical (receiver
// type parameters bound to their owner); a free function through
// Canonical (its own type parameters bound and constrained).
func DeclSignature(fn *types.Func) (string, error) {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return "", fmt.Errorf("typeid: %s has no signature", fn.Name())
	}
	if sig.Recv() != nil {
		return MethodCanonical(fn)
	}
	return Canonical(fn.Type())
}

// idctx carries the recursion-cycle guard, the type-parameter binder
// environment, and a SHARED error slot. Recursive construction records
// the first failure directly through fail(); there is no in-band poison
// marker and no dual "string plus optional check" contract.
type idctx struct {
	seen    map[types.Type]bool
	binders map[*types.TypeParam]string
	errp    *error
}

func newCtx() *idctx {
	return &idctx{seen: map[types.Type]bool{}, binders: map[*types.TypeParam]string{}, errp: new(error)}
}

// fail records the first construction error (an unhandled type form or a
// free parameter with no provable binder).
func (c *idctx) fail(t types.Type) {
	if *c.errp == nil {
		*c.errp = fmt.Errorf("typeid: no exact canonical identity for %s", t.String())
	}
}

// bind records a type parameter at binder position i within the current
// scope. Positions are keyed by the parameter object so distinct binders
// (a function's own vs a receiver's) never alias.
func (c *idctx) bind(tp *types.TypeParam, i int) {
	c.binders[tp] = fmt.Sprintf("$#%d", i)
}

// bindOwner binds a receiver type parameter to its owner-qualified
// identity, so free parameters from distinct generic types never alias.
func (c *idctx) bindOwner(tp *types.TypeParam, owner string, i int) {
	c.binders[tp] = fmt.Sprintf("$%s#%d", owner, i)
}

func (c *idctx) child() *idctx {
	b := make(map[*types.TypeParam]string, len(c.binders))
	for k, v := range c.binders {
		b[k] = v
	}
	return &idctx{seen: c.seen, binders: b, errp: c.errp}
}

func write(out *strings.Builder, t types.Type, c *idctx) {
	t = types.Unalias(t)
	switch u := t.(type) {
	case *types.Basic:
		// rune and byte are predeclared ALIASES (rune≡int32, byte≡uint8):
		// go/types.Identical treats them as identical, so their canonical
		// identity must be the underlying numeric name.
		name := u.Name()
		switch u.Kind() {
		case types.Rune:
			name = "int32"
		case types.Byte:
			name = "uint8"
		}
		out.WriteString(name)
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
				write(out, args.At(i), c)
			}
			out.WriteString("]")
		}
	case *types.Pointer:
		out.WriteString("*")
		write(out, u.Elem(), c)
	case *types.Slice:
		out.WriteString("[]")
		write(out, u.Elem(), c)
	case *types.Array:
		fmt.Fprintf(out, "[%d]", u.Len())
		write(out, u.Elem(), c)
	case *types.Map:
		out.WriteString("map[")
		write(out, u.Key(), c)
		out.WriteString("]")
		write(out, u.Elem(), c)
	case *types.Chan:
		switch u.Dir() {
		case types.SendOnly:
			out.WriteString("chan<- ")
		case types.RecvOnly:
			out.WriteString("<-chan ")
		default:
			out.WriteString("chan ")
		}
		write(out, u.Elem(), c)
	case *types.Signature:
		if c.seen[t] {
			out.WriteString("func(...)")
			return
		}
		c.seen[t] = true
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
		child := c.child()
		if tp := u.TypeParams(); tp != nil && tp.Len() > 0 {
			for i := range tp.Len() {
				child.bind(tp.At(i), i)
			}
			writeTypeParams(out, tp, child)
		}
		writeSignatureBody(out, u, child)
		delete(c.seen, t)
	case *types.Struct:
		if c.seen[t] {
			out.WriteString("struct{...}")
			return
		}
		c.seen[t] = true
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
			write(out, field.Type(), c)
			if tag := u.Tag(i); tag != "" {
				fmt.Fprintf(out, " %q", tag)
			}
		}
		out.WriteString("}")
		delete(c.seen, t)
	case *types.Interface:
		if c.seen[t] {
			out.WriteString("interface{...}")
			return
		}
		c.seen[t] = true
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
			write(&m, method.Type(), c)
			methods = append(methods, m.String())
		}
		sort.Strings(methods)
		for i, m := range methods {
			if i > 0 {
				out.WriteString(";")
			}
			out.WriteString(m)
		}
		// The type-set restriction is the NORMALIZED term set — the
		// INTERSECTION of every embedded element's type set (Go's term
		// algebra), so interface{int|string} ({int,string}) and
		// interface{int; string} (∅) differ while interface{int|string; int}
		// and interface{int} both reduce to {int}.
		terms, comparable, err := interfaceTerms(u, c)
		if err != nil {
			c.fail(t)
		} else if suffix, err := serializeTermset(terms, comparable, c); err != nil {
			c.fail(t)
		} else {
			out.WriteString(suffix)
		}
		out.WriteString("}")
		delete(c.seen, t)
	case *types.TypeParam:
		// A type parameter is identified by its BINDER POSITION within the
		// current environment, never a global index. A parameter with no
		// provable binder (a free receiver parameter serialized without its
		// method context) fails closed.
		if id, ok := c.binders[u]; ok {
			out.WriteString(id)
		} else {
			c.fail(t)
		}
	case *types.Union:
		// A bare constraint union (int | ~string | …): normalized —
		// sorted and de-duplicated — so syntactic order does not matter
		// (union.go: terms are stored non-canonically).
		out.WriteString("union{")
		out.WriteString(strings.Join(dedupeSorted(unionTerms(u, c)), "|"))
		out.WriteString("}")
	case *types.Tuple:
		// Tag arity/boundaries unambiguously: a 1-tuple must NOT serialize
		// like its bare element.
		out.WriteString("tuple(")
		for i := range u.Len() {
			if i > 0 {
				out.WriteString(",")
			}
			write(out, u.At(i).Type(), c)
		}
		out.WriteString(")")
	default:
		// Every well-formed type form is handled above. An unhandled form
		// has no exact identity: fail closed directly (no in-band marker),
		// never a t.String() spelling that does not package-qualify.
		c.fail(t)
	}
}

// writeTypeParams writes a type-parameter list by binder POSITION with
// each parameter's complete constraint type set, so signatures that
// differ only in a constraint have distinct identities while
// alpha-equivalent renamings ([T any] vs [U any]) coincide.
func writeTypeParams(out *strings.Builder, params *types.TypeParamList, c *idctx) {
	if params == nil || params.Len() == 0 {
		return
	}
	out.WriteString("[")
	for i := range params.Len() {
		if i > 0 {
			out.WriteString(",")
		}
		fmt.Fprintf(out, "#%d:", i)
		write(out, params.At(i).Constraint(), c)
	}
	out.WriteString("]")
}

// collectTerms flattens an interface's type-set restriction into its
// terms. Embedded METHOD interfaces contribute no terms (their methods
// are in the complete method set); embedded unions expand to their

// unionTerms renders each union term with tilde approximation preserved.
func unionTerms(u *types.Union, c *idctx) []string {
	terms := make([]string, 0, u.Len())
	for i := range u.Len() {
		term := u.Term(i)
		prefix := ""
		if term.Tilde() {
			prefix = "~"
		}
		terms = append(terms, prefix+canonicalString(term.Type(), c))
	}
	return terms
}

// canonicalString renders one type to its canonical identity string.
func canonicalString(t types.Type, c *idctx) string {
	var b strings.Builder
	write(&b, t, c)
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

// writeSignatureBody writes the callable body func(params)(results) using
// the binder context c (its own type parameters already bound).
func writeSignatureBody(out *strings.Builder, u *types.Signature, c *idctx) {
	out.WriteString("func(")
	params := u.Params()
	for i := range params.Len() {
		if i > 0 {
			out.WriteString(",")
		}
		if u.Variadic() && i == params.Len()-1 {
			out.WriteString("...")
		}
		write(out, params.At(i).Type(), c)
	}
	out.WriteString(")")
	results := u.Results()
	if results.Len() > 0 {
		out.WriteString("(")
		for i := range results.Len() {
			if i > 0 {
				out.WriteString(",")
			}
			write(out, results.At(i).Type(), c)
		}
		out.WriteString(")")
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
