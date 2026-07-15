package emit

import (
	"fmt"
	"go/token"

	"github.com/tsoniclang/gotots/internal/ir"
)

func (p *printer) printBlockBody(block *ir.Block) error {
	for _, stmt := range block.Stmts {
		if err := p.printStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (p *printer) printStmt(stmt ir.Stmt) error {
	switch n := stmt.(type) {
	case *ir.Block:
		p.line("{")
		p.indent++
		if err := p.printBlockBody(n); err != nil {
			return err
		}
		p.indent--
		p.line("}")
		return nil

	case *ir.DeclStmt:
		return p.printDecl(n)

	case *ir.AssignStmt:
		return p.printAssign(n)

	case *ir.IfStmt:
		return p.printIf(n)

	case *ir.ForStmt:
		return p.printFor(n)

	case *ir.ReturnStmt:
		return p.printReturn(n)

	case *ir.ExprStmt:
		call, err := p.printExpr(n.Call)
		if err != nil {
			return err
		}
		p.line("%s;", call)
		return nil

	case *ir.BranchStmt:
		// Inside a range-over-func body, break stops the iteration
		// through the yield protocol and continue yields true; a nested
		// loop or switch that owns the branch cleared the transform.
		if n.Tok == token.BREAK {
			if ctx := p.rangeBreak; ctx != nil {
				p.line("%s = true;", ctx.doneVar)
				p.line("return false;")
				return nil
			}
			p.line("break;")
			return nil
		}
		if ctx := p.rangeContinue; ctx != nil {
			p.line("return true;")
			return nil
		}
		p.line("continue;")
		return nil

	case *ir.TryFinally:
		p.line("try {")
		p.indent++
		if err := p.printBlockBody(n.Body); err != nil {
			return err
		}
		p.indent--
		p.line("} finally {")
		p.indent++
		if err := p.printBlockBody(n.Finally); err != nil {
			return err
		}
		p.indent--
		p.line("}")
		return nil

	case *ir.RangeSlice:
		return p.printRangeSlice(n)

	case *ir.RangeString:
		return p.printRangeString(n)

	case *ir.RangeMap:
		return p.printRangeMap(n)

	case *ir.RangeFunc:
		return p.printRangeFunc(n)

	case *ir.DeferPush:
		call, err := p.printExpr(n.Call)
		if err != nil {
			return err
		}
		p.line("_ds$.push(() => { %s; });", call)
		return nil

	case *ir.SwitchStmt:
		return p.printSwitch(n)

	case *ir.TypeSwitchStmt:
		return p.printTypeSwitch(n)

	case *ir.MapDeleteStmt:
		mapExpr, err := p.printExpr(n.Map)
		if err != nil {
			return err
		}
		key, err := p.printExpr(n.Key)
		if err != nil {
			return err
		}
		p.line("gort$.%s(%s, %s);", mapHelper("goMapDelete", n.Map), mapExpr, key)
		return nil

	case *ir.MapClearStmt:
		mapExpr, err := p.printExpr(n.Map)
		if err != nil {
			return err
		}
		p.line("gort$.%s(%s);", mapHelper("goMapClear", n.Map), mapExpr)
		return nil

	case *ir.PanicStmt:
		value, err := p.printExpr(n.Value)
		if err != nil {
			return err
		}
		p.line("gort$.goPanicValue(%s);", value)
		return nil

	case *ir.RangeInt:
		return p.printRangeInt(n)
	}
	return fmt.Errorf("no emission for IR statement %T", stmt)
}

// printLoopBody prints a nested loop's body: the loop owns break and
// continue, so both range-over-func branch transforms clear.
func (p *printer) printLoopBody(block *ir.Block) error {
	savedBreak, savedContinue := p.rangeBreak, p.rangeContinue
	p.rangeBreak, p.rangeContinue = nil, nil
	err := p.printBlockBody(block)
	p.rangeBreak, p.rangeContinue = savedBreak, savedContinue
	return err
}

// printSwitchClauseBody prints a switch clause body: the switch owns
// break, but continue still targets the enclosing loop.
func (p *printer) printSwitchClauseBody(block *ir.Block) error {
	savedBreak := p.rangeBreak
	p.rangeBreak = nil
	err := p.printBlockBody(block)
	p.rangeBreak = savedBreak
	return err
}

func (p *printer) printDecl(n *ir.DeclStmt) error {
	if n.Tuple != nil {
		tupleValue, err := p.printExpr(n.Tuple)
		if err != nil {
			return err
		}
		tuple := p.temp()
		p.line("const %s = %s;", tuple, tupleValue)
		for i, name := range n.Names {
			if name == "_" {
				continue // discarded slot; the tuple was evaluated once
			}
			if i < len(n.Reused) && n.Reused[i] {
				// := reassigns this existing name: a whole-value store
				// with the same in-place semantics as any assignment.
				staged := stagedTarget{kind: "var", name: tsName(name),
					structValue: n.Types[i].Kind == ir.KindStruct}
				var err error
				if staged.arrayValue, err = p.arrayValueCallback(n.Types[i]); err != nil {
					return err
				}
				if err := staged.store(p, fmt.Sprintf("%s[%d]", tuple, i)); err != nil {
					return err
				}
				continue
			}
			spelled, err := p.tsType(n.Types[i])
			if err != nil {
				return err
			}
			// A struct or array slot binds a value copy (a comma-ok
			// lookup yields the stored instance).
			if n.Types[i].Kind == ir.KindStruct {
				p.line("let %s: %s = %s[%d].goClone$();", tsName(name), spelled, tuple, i)
				continue
			}
			if n.Types[i].Kind == ir.KindArray {
				cloneElem, err := p.arrayElemClone(*n.Types[i].Elem)
				if err != nil {
					return err
				}
				p.line("let %s: %s = gosl$.goArrayClone(%s[%d], %s);", tsName(name), spelled, tuple, i, cloneElem)
				continue
			}
			if t := n.Types[i]; t.Kind == ir.KindExternal {
				p.line("let %s: %s = goext$.goExternalCall(%q, [%s[%d]]) as %s;",
					tsName(name), spelled, t.Pkg+"."+t.Named+".goClone$", tuple, i, spelled)
				continue
			}
			p.line("let %s: %s = %s[%d];", tsName(name), spelled, tuple, i)
		}
		return nil
	}
	for i, name := range n.Names {
		value, err := p.printExpr(n.Values[i])
		if err != nil {
			return err
		}
		if name == "_" {
			// Go evaluates discarded values.
			p.line("void (%s);", value)
			continue
		}
		spelled, err := p.tsType(n.Types[i])
		if err != nil {
			return err
		}
		p.line("let %s: %s = %s;", tsName(name), spelled, value)
	}
	return nil
}

func (p *printer) printAssign(n *ir.AssignStmt) error {
	if n.Tuple != nil {
		// Go's usual order is lexical: the left-hand operands (pointer
		// bases, map operands, keys) evaluate before the right-hand
		// tuple expression, and the stores happen last.
		staged := make([]stagedTarget, len(n.Targets))
		for i, target := range n.Targets {
			var err error
			staged[i], err = p.stageTarget(target)
			if err != nil {
				return err
			}
		}
		tupleValue, err := p.printExpr(n.Tuple)
		if err != nil {
			return err
		}
		tuple := p.temp()
		p.line("const %s = %s;", tuple, tupleValue)
		for i := range n.Targets {
			if err := staged[i].store(p, fmt.Sprintf("%s[%d]", tuple, i)); err != nil {
				return err
			}
		}
		return nil
	}
	if len(n.Targets) == 1 {
		value, err := p.printExpr(n.Values[0])
		if err != nil {
			return err
		}
		return p.printStore(n.Targets[0], value)
	}
	// Go's two-phase rule: target operands (pointer bases, map operands
	// and keys) and right-hand values are evaluated in source order into
	// temporaries first, then every store happens.
	staged := make([]stagedTarget, len(n.Targets))
	for i, target := range n.Targets {
		var err error
		staged[i], err = p.stageTarget(target)
		if err != nil {
			return err
		}
	}
	temps := make([]string, len(n.Values))
	for i, value := range n.Values {
		printed, err := p.printExpr(value)
		if err != nil {
			return err
		}
		temps[i] = p.temp()
		p.line("const %s = %s;", temps[i], printed)
	}
	for i := range n.Targets {
		if err := staged[i].store(p, temps[i]); err != nil {
			return err
		}
	}
	return nil
}

// printSwitch emits a Go expression switch as a JS switch, whose
// semantics coincide exactly for the reviewed carriers: the tag is
// evaluated once, case expressions are evaluated in source order under
// identity equality, the first match wins, and default runs only when
// nothing matches. Each clause body gets its own block for Go's per-case
// scoping; an emitted break is Go's implicit clause exit, and omitting it
// is Go's fallthrough.
func (p *printer) printSwitch(n *ir.SwitchStmt) error {
	if n.Init != nil {
		p.line("{")
		p.indent++
		if err := p.printStmt(n.Init); err != nil {
			return err
		}
	}
	tag, err := p.printExpr(n.Tag)
	if err != nil {
		return err
	}
	p.line("switch (%s) {", tag)
	p.indent++
	for _, clause := range n.Clauses {
		if clause.Values == nil {
			p.line("default: {")
		} else {
			for i, value := range clause.Values {
				printed, err := p.printExpr(value)
				if err != nil {
					return err
				}
				if i < len(clause.Values)-1 {
					p.line("case %s:", printed)
				} else {
					p.line("case %s: {", printed)
				}
			}
		}
		p.indent++
		if err := p.printSwitchClauseBody(clause.Body); err != nil {
			return err
		}
		if !clause.Fallthrough {
			p.line("break;")
		}
		p.indent--
		p.line("}")
	}
	p.indent--
	p.line("}")
	if n.Init != nil {
		p.indent--
		p.line("}")
	}
	return nil
}

func (p *printer) printIf(n *ir.IfStmt) error {
	// A Go if-init introduces its own scope covering both branches.
	if n.Init != nil {
		p.line("{")
		p.indent++
		if err := p.printStmt(n.Init); err != nil {
			return err
		}
	}
	cond, err := p.printExpr(n.Cond)
	if err != nil {
		return err
	}
	p.line("if (%s) {", cond)
	p.indent++
	if err := p.printBlockBody(n.Then); err != nil {
		return err
	}
	p.indent--
	switch elseStmt := n.Else.(type) {
	case nil:
		p.line("}")
	case *ir.Block:
		p.line("} else {")
		p.indent++
		if err := p.printBlockBody(elseStmt); err != nil {
			return err
		}
		p.indent--
		p.line("}")
	case *ir.IfStmt:
		// Emit as a nested else-block; init scoping stays correct.
		p.line("} else {")
		p.indent++
		if err := p.printStmt(elseStmt); err != nil {
			return err
		}
		p.indent--
		p.line("}")
	default:
		return fmt.Errorf("no emission for else branch %T", n.Else)
	}
	if n.Init != nil {
		p.indent--
		p.line("}")
	}
	return nil
}

// printFor emits the classic three-clause loop as a JS for header so that
// continue still executes the post statement (a body-tail post would be
// skipped by continue).
func (p *printer) printFor(n *ir.ForStmt) error {
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
	p.line("for (%s; %s; %s) {", init, cond, post)
	p.indent++
	if err := p.printLoopBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// forClause renders an init/post statement as a for-header clause.
func (p *printer) forClause(stmt ir.Stmt, isInit bool) (string, error) {
	switch n := stmt.(type) {
	case nil:
		return "", nil
	case *ir.DeclStmt:
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
		if n.Tuple != nil || len(n.Targets) != 1 {
			return "", fmt.Errorf("assignment not expressible in this for-loop clause")
		}
		variable, isVar := n.Targets[0].(ir.VarTarget)
		if !isVar {
			return "", fmt.Errorf("non-variable assignment in a for-loop clause")
		}
		value, err := p.printExpr(n.Values[0])
		if err != nil {
			return "", err
		}
		if variable.T.Kind == ir.KindStruct {
			return tsName(variable.Name) + ".goSet$(" + value + ")", nil
		}
		if variable.T.Kind == ir.KindArray {
			setElem, err := p.arrayElemSet(*variable.T.Elem)
			if err != nil {
				return "", err
			}
			return "gosl$.goArraySetAll(" + tsName(variable.Name) + ", " + value + ", " + setElem + ")", nil
		}
		return tsName(variable.Name) + " = " + value, nil
	}
	return "", fmt.Errorf("statement %T not expressible in a for-loop clause", stmt)
}

func (p *printer) printReturn(n *ir.ReturnStmt) error {
	if n.CallValue != nil {
		call, err := p.printExpr(n.CallValue)
		if err != nil {
			return err
		}
		p.emitFunctionReturn(call)
		return nil
	}
	switch len(n.Values) {
	case 0:
		p.emitFunctionReturn("")
	case 1:
		value, err := p.printExpr(n.Values[0])
		if err != nil {
			return err
		}
		p.emitFunctionReturn(value)
	default:
		parts := make([]string, len(n.Values))
		for i, value := range n.Values {
			printed, err := p.printExpr(value)
			if err != nil {
				return err
			}
			parts[i] = printed
		}
		p.emitFunctionReturn("[" + joinComma(parts) + "]")
	}
	return nil
}

// emitFunctionReturn returns from the enclosing Go function: directly,
// or — inside a range-over-func yield closure — by capturing the value,
// stopping the iteration, and letting the loop re-issue the return
// after the sequence function unwinds. The empty string is a bare
// return.
func (p *printer) emitFunctionReturn(value string) {
	if ctx := p.rangeReturn; ctx != nil {
		if value != "" {
			p.line("%s = %s;", ctx.retVar, value)
		}
		p.line("%s = true;", ctx.returnedVar)
		p.line("%s = true;", ctx.doneVar)
		p.line("return false;")
		return
	}
	if value == "" {
		p.line("return;")
		return
	}
	p.line("return %s;", value)
}

func joinComma(parts []string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ", "
		}
		out += part
	}
	return out
}
