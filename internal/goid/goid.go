// Package goid defines the canonical Go declaration identity used as the
// proof-chain key by the census, the translator, and every downstream
// ledger. Identity derives from Go package and object identity, never file
// layout: moving a declaration between files of one package does not
// change its identity, so cross-pin deltas compare real semantic changes.
//
// The only declarations Go permits to repeat in package scope — package-
// level func init and blank identifiers — are qualified by their declaring
// file and position, which is exactly the evidence that distinguishes
// them.
package goid

import "go/types"

import "fmt"

// Func is the identity of a package-level function.
func Func(pkgPath, name string) string {
	return pkgPath + "::func::" + name
}

// Method is the identity of a method on a named receiver type.
func Method(pkgPath, receiver, name string) string {
	return pkgPath + "::method::" + receiver + "." + name
}

// Value is the identity of a package-level const or var
// (kind is "const" or "var").
func Value(pkgPath, kind, name string) string {
	return pkgPath + "::" + kind + "::" + name
}

// TypeName is the identity of a named type or alias
// (kind is "type" or "alias").
func TypeName(pkgPath, kind, name string) string {
	return pkgPath + "::" + kind + "::" + name
}

// Repeatable qualifies one of Go's legally repeatable declarations —
// package-level func init or a blank identifier — by its declaring file
// and exact position.
func Repeatable(pkgPath, kind, name, file string, line, col int) string {
	return fmt.Sprintf("%s::%s::%s@%s:%d:%d", pkgPath, kind, name, file, line, col)
}

// IsRepeatable reports whether a declaration name needs position
// qualification: blank identifiers always, and init only as a plain
// function (methods named init cannot repeat).
func IsRepeatable(kind, name string) bool {
	if name == "_" {
		return true
	}
	return kind == "func" && name == "init"
}

// CanonicalReceiver resolves a method's receiver base name through the
// TYPE OBJECT — aliases resolve to the canonical named type, so one
// method has one identity regardless of how a declaration spells its
// receiver. The syntactic spelling is never identity.
func CanonicalReceiver(method *types.Func) string {
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return ""
	}
	t := signature.Recv().Type()
	if pointer, isPointer := t.(*types.Pointer); isPointer {
		t = pointer.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return ""
	}
	return named.Obj().Name()
}
