// Defer lowering: top-level defers nest as try/finally; a body with
// defers below the top level switches uniformly onto a per-function
// LIFO defer stack drained at every exit. Receivers and arguments are
// captured at the defer site in both forms.
package ir

import (
	"fmt"
	"go/ast"
)

// hasNestedDefer reports whether any defer statement sits below the
// top-level statement list (function literals keep their own defers).
func hasNestedDefer(stmts []ast.Stmt) bool {
	nested := false
	for _, stmt := range stmts {
		if _, isDefer := stmt.(*ast.DeferStmt); isDefer {
			continue // top-level defer
		}
		ast.Inspect(stmt, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.FuncLit:
				return false
			case *ast.DeferStmt:
				nested = true
			}
			return !nested
		})
		if nested {
			return true
		}
	}
	return false
}

// buildDeferredCall captures the deferred call's receiver and arguments at
// the defer site (Go evaluates them when the defer statement executes) and
// returns the capture declarations plus the call over the captures.
func (b *builder) buildDeferredCall(deferStmt *ast.DeferStmt) ([]Stmt, Expr, error) {
	span := b.span(deferStmt.Pos())
	built, err := b.buildAnyCall(deferStmt.Call)
	if err != nil {
		return nil, nil, err
	}
	prefix := fmt.Sprintf("_d%d$", b.deferCount)
	b.deferCount++

	var captures []Stmt
	capture := func(name string, value Expr) *VarRef {
		captures = append(captures, &DeclStmt{
			Names:  []string{name},
			Types:  []Type{value.Type()},
			Values: []Expr{value},
		})
		return &VarRef{Name: name, T: value.Type()}
	}

	switch call := built.(type) {
	case *Call:
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *MethodCall:
		// A value receiver copies when the defer statement evaluates —
		// mutations between the defer and the call never reach it — so a
		// struct-valued receiver captures a clone.
		recv := call.Recv
		if !call.PointerRecv {
			recv = b.bindStructValue(recv)
		}
		call.Recv = capture(prefix+"_r", recv)
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *DynCall:
		// The function value is captured at the defer site; a nil value
		// panics when invoked, not when deferred — exactly Go.
		call.Fun = capture(prefix+"_f", call.Fun)
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *ExternalCall:
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *ExternalMethodCall:
		recv := call.Recv
		if !call.PointerRecv {
			recv = b.bindStructValue(recv)
		}
		call.Recv = capture(prefix+"_r", recv)
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *IfaceCall:
		call.Recv = capture(prefix+"_r", call.Recv)
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	}
	return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "deferred non-call expression", Span: span}
}
