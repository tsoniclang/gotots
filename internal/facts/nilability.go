package facts

import (
	"go/ast"
	"go/token"
)

// AnalyzeReceiverNilability is the ADR-0006 producer: for one method
// body it decides whether a call-site nil check is OBSERVATIONALLY
// EQUIVALENT to the first in-body receiver dereference — no effect,
// observable call, mutation of non-local state, or defer/recover
// boundary can execute before that dereference on any path that
// reaches it. The analysis is conservative-exact: a proof failure
// keeps the method on its current exact lowering; only proven methods
// move to ordinary class-method emission.
//
// A receiver compared against nil anywhere before the proof resolves
// marks the method nil-TOLERANT (calibration class B7): it is
// semantically defined for nil receivers and takes the recorded
// free-function exception, never an entry check.
func AnalyzeReceiverNilability(recvName string, pointerReceiver bool, body *ast.BlockStmt) ReceiverNilability {
	if !pointerReceiver {
		// A value receiver cannot be nil; ordinary emission needs no
		// check and the question is vacuous.
		return ReceiverNilability{EquivalentAtEntry: true}
	}
	if body == nil || recvName == "" || recvName == "_" {
		// A bodiless or receiver-less method never dereferences; nil
		// tolerance is trivial.
		return ReceiverNilability{ToleratesNil: true}
	}
	a := &nilabilityWalk{recv: recvName}
	switch a.stmts(body.List) {
	case stProved:
		return ReceiverNilability{EquivalentAtEntry: true}
	case stTolerant:
		return ReceiverNilability{ToleratesNil: true}
	default:
		return ReceiverNilability{}
	}
}

// status is the tri-state (plus tolerance) result of scanning a
// construct in evaluation order.
type status int

const (
	// stContinue: nothing observable happened; scanning continues.
	stContinue status = iota
	// stProved: the first observable event was a receiver dereference.
	stProved
	// stFailed: an observable event (or an unordered construct)
	// precedes any dereference on some path.
	stFailed
	// stTolerant: the receiver was compared with nil — the method
	// treats nil as a value.
	stTolerant
)

type nilabilityWalk struct {
	recv string
}

func (a *nilabilityWalk) isReceiver(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == a.recv
}

// stmts joins a statement sequence: the first non-Continue status
// resolves the sequence.
func (a *nilabilityWalk) stmts(list []ast.Stmt) status {
	for _, stmt := range list {
		if result := a.stmt(stmt); result != stContinue {
			return result
		}
	}
	return stContinue
}

// join combines two exclusive branch paths whose successor is the same
// statement sequence.
func join(then, other status) status {
	switch {
	case then == stTolerant || other == stTolerant:
		return stTolerant
	case then == stFailed || other == stFailed:
		return stFailed
	case then == stProved && other == stProved:
		return stProved
	case then == stContinue && other == stContinue:
		return stContinue
	default:
		// One arm proved, the other continues effect-free: the proved
		// path is safe and the continuing path defers to the successor
		// statements.
		return stContinue
	}
}

func (a *nilabilityWalk) stmt(stmt ast.Stmt) status {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return a.expr(s.X)
	case *ast.IfStmt:
		if s.Init != nil {
			if result := a.stmt(s.Init); result != stContinue {
				return result
			}
		}
		if result := a.expr(s.Cond); result != stContinue {
			return result
		}
		thenStatus := a.stmts(s.Body.List)
		elseStatus := stContinue
		if s.Else != nil {
			switch e := s.Else.(type) {
			case *ast.BlockStmt:
				elseStatus = a.stmts(e.List)
			case *ast.IfStmt:
				elseStatus = a.stmt(e)
			}
		}
		return join(thenStatus, elseStatus)
	case *ast.AssignStmt:
		// RHS evaluates first, in order.
		for _, rhs := range s.Rhs {
			if result := a.expr(rhs); result != stContinue {
				return result
			}
		}
		if s.Tok == token.DEFINE {
			return stContinue
		}
		for _, lhs := range s.Lhs {
			switch target := lhs.(type) {
			case *ast.Ident:
				// A store to a local (or blank) is unobservable.
				if target.Name == "_" || target.Obj != nil {
					continue
				}
				return stFailed
			case *ast.SelectorExpr:
				if a.isReceiver(target.X) {
					// Storing THROUGH the receiver dereferences it.
					return stProved
				}
				return stFailed
			case *ast.IndexExpr:
				if a.isReceiver(target.X) {
					return stProved
				}
				return stFailed
			default:
				return stFailed
			}
		}
		return stContinue
	case *ast.IncDecStmt:
		if selector, ok := s.X.(*ast.SelectorExpr); ok && a.isReceiver(selector.X) {
			return stProved
		}
		if ident, ok := s.X.(*ast.Ident); ok && ident.Obj != nil {
			return stContinue
		}
		return stFailed
	case *ast.DeclStmt:
		if decl, ok := s.Decl.(*ast.GenDecl); ok {
			for _, spec := range decl.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok {
					for _, expr := range value.Values {
						if result := a.expr(expr); result != stContinue {
							return result
						}
					}
				}
			}
		}
		return stContinue
	case *ast.ReturnStmt:
		for _, result := range s.Results {
			if status := a.expr(result); status != stContinue {
				return status
			}
		}
		// A return before any dereference: this path never
		// dereferences — an entry check would panic where Go does not.
		return stFailed
	case *ast.BlockStmt:
		return a.stmts(s.List)
	case *ast.ForStmt:
		if s.Init != nil {
			if result := a.stmt(s.Init); result != stContinue {
				return result
			}
		}
		if s.Cond != nil {
			if result := a.expr(s.Cond); result != stContinue {
				return result
			}
		}
		// The body may run zero times; effects inside cannot be ordered
		// against the zero-iteration continuation by this pass.
		return stFailed
	case *ast.RangeStmt:
		// The range expression evaluates exactly once, first.
		if result := a.expr(s.X); result != stContinue {
			return result
		}
		return stFailed
	case *ast.SwitchStmt:
		if s.Init != nil {
			if result := a.stmt(s.Init); result != stContinue {
				return result
			}
		}
		if s.Tag != nil {
			if result := a.expr(s.Tag); result != stContinue {
				return result
			}
		}
		return stFailed
	case *ast.TypeSwitchStmt:
		return stFailed
	default:
		// Defer/go/send/select/labels and anything unmodeled: not
		// ordered exactly by this conservative pass.
		return stFailed
	}
}

func (a *nilabilityWalk) expr(expr ast.Expr) status {
	if expr == nil {
		return stContinue
	}
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if a.isReceiver(e.X) {
			// recv.field / recv.method — the pointer dereference.
			return stProved
		}
		return a.expr(e.X)
	case *ast.StarExpr:
		if a.isReceiver(e.X) {
			return stProved
		}
		return a.expr(e.X)
	case *ast.IndexExpr:
		if a.isReceiver(e.X) {
			return stProved
		}
		if result := a.expr(e.X); result != stContinue {
			return result
		}
		return a.expr(e.Index)
	case *ast.BinaryExpr:
		// Receiver-nil comparison: the method treats nil as a value.
		if (e.Op == token.EQL || e.Op == token.NEQ) &&
			(a.isReceiver(e.X) && isNil(e.Y) || a.isReceiver(e.Y) && isNil(e.X)) {
			return stTolerant
		}
		if result := a.expr(e.X); result != stContinue {
			return result
		}
		return a.expr(e.Y)
	case *ast.ParenExpr:
		return a.expr(e.X)
	case *ast.UnaryExpr:
		return a.expr(e.X)
	case *ast.CallExpr:
		// recv.Method(...) dereferences the receiver to dispatch.
		if selector, ok := e.Fun.(*ast.SelectorExpr); ok && a.isReceiver(selector.X) {
			return stProved
		}
		// Arguments evaluate before the call; a dereference inside an
		// argument still proves. The call itself is observable.
		for _, arg := range e.Args {
			if result := a.expr(arg); result != stContinue {
				return result
			}
		}
		return stFailed
	case *ast.Ident, *ast.BasicLit, *ast.FuncLit:
		// Reading a local or literal, or constructing a closure without
		// calling it, is unobservable.
		return stContinue
	case *ast.CompositeLit:
		for _, element := range e.Elts {
			if result := a.expr(element); result != stContinue {
				return result
			}
		}
		return stContinue
	case *ast.KeyValueExpr:
		return a.expr(e.Value)
	case *ast.TypeAssertExpr:
		return a.expr(e.X)
	default:
		return stFailed
	}
}

func isNil(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}
