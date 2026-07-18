// For-loop lowering: a native JS for-header for simple single-
// expression clauses, and a normalized while-form (reusing the shared
// declaration/assignment emitters) for multi-result or otherwise
// complex init/post — preserving Go's readonly-tuple typing, value
// copies, one-shot evaluation, simultaneous assignment, and
// continue-runs-post exactly once.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printFor emits the classic three-clause loop. When both the init and
// post render as simple single-expression clauses it uses a native JS
// for-header, so continue still executes the post (a body-tail post would
// be skipped by continue). Otherwise it lowers to a normalized while-form
// that emits the init and post through the shared declaration/assignment
// emitters — the single place Go's full binding and store semantics live.
func (p *printer) printFor(n *ir.ForStmt) error {
	if !p.forClauseable(n.Init, true) || !p.forClauseable(n.Post, false) {
		return p.printForNormalized(n)
	}
	init, err := p.forClause(n.Init, true)
	if err != nil {
		return err
	}
	cond := ""
	if n.Cond != nil {
		cond, err = p.printExpr(n.Cond)
		if err != nil {
			return err
		}
	}
	post, err := p.forClause(n.Post, false)
	if err != nil {
		return err
	}
	p.line("%sfor (%s; %s; %s) {", p.takeLoopLabel(), init, cond, post)
	p.indent++
	if err := p.printLoopBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// forClauseable reports whether an init/post statement renders as a
// single-expression for-header clause. Only these stay a native
// for-header; every other shape (multi-result init or assignment,
// parallel or non-variable assignment, native-slice whole-value store, or
// any other statement) is lowered by printForNormalized through the
// shared declaration/assignment emitters, which preserve Go's readonly
// tuple typing, value copies, staged pointer/boxed stores, reused and
// blank slots, and simultaneous assignment exactly.
func (p *printer) forClauseable(stmt ir.Stmt, isInit bool) bool {
	switch n := stmt.(type) {
	case nil:
		return true
	case *ir.DeclStmt:
		if !isInit || n.Tuple != nil {
			return false
		}
		for _, name := range n.Names {
			if p.slicePlans[name] == "native-array" {
				return false // printNativeSliceDecl form
			}
		}
		return true
	case *ir.AssignStmt:
		if n.Tuple != nil || len(n.Targets) != 1 {
			return false
		}
		return p.nativeStorable(n.Targets[0])
	case *ir.CompoundStmt:
		switch n.Target.(type) {
		case ir.VarTarget, ir.BoxedTarget:
			return true
		}
		return false
	}
	return false
}

// nativeStorable reports whether a target's store is a single expression
// with no pre-evaluated operands, nil checks, or bounds panics — so it can
// stand as a native for-header clause. Only a plain variable (not a
// native-array local slice, whose whole-value store is a distinct
// construction), a boxed cell, and a blank qualify; every field, pointee,
// map, and slice element needs staged operands and is normalized. This is
// the eligibility half of the store plan (which targets), kept separate
// from the store-semantics half (how each carrier stores), which lives in
// the shared varStoreExpr — so external is never a special case here.
func (p *printer) nativeStorable(target ir.Target) bool {
	switch t := target.(type) {
	case ir.BlankTarget, ir.BoxedTarget:
		return true
	case ir.VarTarget:
		if t.Pkg == "" && t.T.Kind == ir.KindSlice && p.slicePlans[t.Name] == "native-array" {
			return false // printNativeSliceAssign form
		}
		return true
	}
	return false
}

// storeClauseExpr renders a native-storable target's store as a single
// clause expression, reusing the shared varStoreExpr for variables so the
// clause and the statement store are byte-identical by construction.
func (p *printer) storeClauseExpr(target ir.Target, value string) (string, error) {
	switch t := target.(type) {
	case ir.BlankTarget:
		return "void (" + value + ")", nil
	case ir.BoxedTarget:
		return t.Cell + ".v = " + value, nil
	case ir.VarTarget:
		return p.varStoreExpr(t, value)
	}
	return "", fmt.Errorf("assignment target %T not expressible in a for-loop clause", target)
}

// forClause renders an init/post statement as a for-header clause.
func (p *printer) forClause(stmt ir.Stmt, isInit bool) (string, error) {
	switch n := stmt.(type) {
	case nil:
		return "", nil
	case *ir.DeclStmt:
		// A multi-result initializer is lowered by printForNormalized;
		// forClause only renders simple single-value declarations (guarded
		// by forClauseable).
		if !isInit || n.Tuple != nil {
			return "", fmt.Errorf("declaration not expressible in this for-loop clause")
		}
		parts := make([]string, len(n.Names))
		for i, name := range n.Names {
			spelled, err := p.tsType(n.Types[i])
			if err != nil {
				return "", err
			}
			value, err := p.printExpr(n.Values[i])
			if err != nil {
				return "", err
			}
			parts[i] = fmt.Sprintf("%s: %s = %s", tsName(name), spelled, value)
		}
		return "let " + joinComma(parts), nil
	case *ir.AssignStmt:
		// Multi-result and parallel assignments are lowered by
		// printForNormalized (which stages every target and stores after a
		// single right-hand evaluation); forClause renders a single
		// native-storable target (guarded by forClauseable) through the SAME
		// store rendering as the statement store, so a native header can
		// never diverge from the normalized form — an external value stores
		// in place from both, never rebinds.
		if n.Tuple != nil || len(n.Targets) != 1 {
			return "", fmt.Errorf("assignment not expressible in this for-loop clause")
		}
		value, err := p.printExpr(n.Values[0])
		if err != nil {
			return "", err
		}
		return p.storeClauseExpr(n.Targets[0], value)
	case *ir.CompoundStmt:
		// Simple variable and cell targets evaluate once trivially, so
		// the clause form needs no staging.
		var location string
		switch target := n.Target.(type) {
		case ir.VarTarget:
			location = tsName(target.Name)
		case ir.BoxedTarget:
			location = target.Cell + ".v"
		default:
			return "", fmt.Errorf("compound target %T not expressible in a for-loop clause", n.Target)
		}
		rhs, err := p.printExpr(n.Rhs)
		if err != nil {
			return "", err
		}
		value, err := p.printBinaryOp(n.Op, location, rhs, n.OperandT, n.Rhs.Type().Kind)
		if err != nil {
			return "", err
		}
		return location + " = " + value, nil
	}
	return "", fmt.Errorf("statement %T not expressible in a for-loop clause", stmt)
}

// printForNormalized lowers a for-loop whose init or post is not a simple
// single-expression clause into a while-form:
//
//	{
//	  <INIT>
//	  brk$N: while (true) {
//	    if (!(COND)) break brk$N;
//	    cont$N: { <BODY> }
//	    <POST>
//	  }
//	}
//
// The INIT and POST run through the shared statement emitters, so a
// multi-result init binds via readonly-tuple indexing (no mutable-tuple
// annotation) and a multi-result/parallel post stages every target and
// stores after one right-hand evaluation — Go's binding, value-copy, and
// simultaneous-assignment semantics, reused across the language. The
// outer block scopes the init-declared loop variables; continue redirects
// to `break cont$N` (leaving the body block, so POST still runs exactly
// once) and break to `break brk$N`, preserving native for-header timing.
func (p *printer) printForNormalized(n *ir.ForStmt) error {
	goLabel := p.pendingLoopLabel
	p.pendingLoopLabel = ""
	p.loopSeq++
	ctx := &normLoopCtx{
		breakLabel: fmt.Sprintf("brk$%d", p.loopSeq),
		contLabel:  fmt.Sprintf("cont$%d", p.loopSeq),
	}
	p.line("{")
	p.indent++
	if n.Init != nil {
		if err := p.printStmt(n.Init); err != nil {
			return err
		}
	}
	p.line("%s: while (true) {", ctx.breakLabel)
	p.indent++
	if n.Cond != nil {
		cond, err := p.printExpr(n.Cond)
		if err != nil {
			return err
		}
		p.line("if (!(%s)) break %s;", cond, ctx.breakLabel)
	}
	p.line("%s: {", ctx.contLabel)
	p.indent++
	if err := p.printNormalizedLoopBody(n.Body, ctx, goLabel); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	if n.Post != nil {
		if err := p.printStmt(n.Post); err != nil {
			return err
		}
	}
	p.indent--
	p.line("}")
	p.indent--
	p.line("}")
	return nil
}

// printNormalizedLoopBody prints a normalized loop's body with break and
// continue redirected to the loop's labels. The loop owns both branches,
// so the range-over-func transforms clear; a Go label on the loop maps to
// this context so a labeled branch targeting it redirects as well.
func (p *printer) printNormalizedLoopBody(block *ir.Block, ctx *normLoopCtx, goLabel string) error {
	savedRangeBreak, savedRangeContinue := p.rangeBreak, p.rangeContinue
	savedNormBreak, savedNormContinue := p.normBreak, p.normContinue
	p.rangeBreak, p.rangeContinue = nil, nil
	p.normBreak, p.normContinue = ctx, ctx
	var restoreLabel func()
	if goLabel != "" {
		if p.normLabels == nil {
			p.normLabels = map[string]*normLoopCtx{}
		}
		prev, had := p.normLabels[goLabel]
		p.normLabels[goLabel] = ctx
		restoreLabel = func() {
			if had {
				p.normLabels[goLabel] = prev
			} else {
				delete(p.normLabels, goLabel)
			}
		}
	}
	err := p.printBlockBody(block)
	p.rangeBreak, p.rangeContinue = savedRangeBreak, savedRangeContinue
	p.normBreak, p.normContinue = savedNormBreak, savedNormContinue
	if restoreLabel != nil {
		restoreLabel()
	}
	return err
}
