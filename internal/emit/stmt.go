package emit

import (
	"fmt"
	"go/token"
	"strings"

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
		if n.Label != "" {
			// A label targeting a normalized loop redirects to its
			// while / body-block labels; otherwise Go's labeled
			// loop/switch branches coincide exactly with JS labeled
			// statements (yield-boundary crossings fail closed at build).
			if ctx := p.normLabels[labelName(n.Label)]; ctx != nil {
				if n.Tok == token.BREAK {
					p.line("break %s;", ctx.breakLabel)
				} else {
					p.line("break %s;", ctx.contLabel)
				}
				return nil
			}
			if n.Tok == token.BREAK {
				p.line("break %s;", labelName(n.Label))
			} else {
				p.line("continue %s;", labelName(n.Label))
			}
			return nil
		}
		// A normalized loop owns the innermost break/continue and
		// redirects to its labels so the post statement still runs.
		// Otherwise, inside a range-over-func body, break stops the
		// iteration through the yield protocol and continue yields true;
		// a nested loop or switch that owns the branch cleared both
		// transforms.
		if n.Tok == token.BREAK {
			if ctx := p.normBreak; ctx != nil {
				p.line("break %s;", ctx.breakLabel)
				return nil
			}
			if ctx := p.rangeBreak; ctx != nil {
				p.line("%s = true;", ctx.doneVar)
				p.line("return false;")
				return nil
			}
			p.line("break;")
			return nil
		}
		if ctx := p.normContinue; ctx != nil {
			p.line("break %s;", ctx.contLabel)
			return nil
		}
		if ctx := p.rangeContinue; ctx != nil {
			p.line("return true;")
			return nil
		}
		p.line("continue;")
		return nil

	case *ir.LabeledStmt:
		p.pendingLoopLabel = labelName(n.Label)
		err := p.printStmt(n.Stmt)
		p.pendingLoopLabel = ""
		return err

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

	case *ir.StmtSeq:
		for _, inner := range n.Stmts {
			if err := p.printStmt(inner); err != nil {
				return err
			}
		}
		return nil

	case *ir.BoxedDecl:
		init, err := p.printExpr(n.Init)
		if err != nil {
			return err
		}
		spelled, err := p.tsType(n.Elem)
		if err != nil {
			return err
		}
		p.line("const %s: gort$.GoCell<%s> = { v: %s };", n.Cell, spelled, init)
		return nil

	case *ir.CompoundStmt:
		// gc's order: impure target operands (calls) evaluate first at
		// their lexical position; then the RHS; then the pure container /
		// base / pointer is read; then the store. Impure operand temps
		// print inside stageCompoundTarget BEFORE the RHS temp below.
		var impure strings.Builder
		operandPrinter := &printer{out: &impure, module: p.module, indent: p.indent,
			temps: p.temps, zeroFactories: p.zeroFactories, eqOps: p.eqOps,
			cloneOps: p.cloneOps, setOps: p.setOps, slicePlans: p.slicePlans}
		staged, err := operandPrinter.stageCompoundTarget(n.Target)
		if err != nil {
			return err
		}
		p.temps = operandPrinter.temps
		p.out.WriteString(impure.String())
		rhs, err := p.printExpr(n.Rhs)
		if err != nil {
			return err
		}
		rhsTemp := p.temp()
		p.line("const %s = %s;", rhsTemp, rhs)
		loaded, err := staged.load(p, n.OperandT)
		if err != nil {
			return err
		}
		loadTemp := p.temp()
		p.line("const %s = %s;", loadTemp, loaded)
		value, err := p.printBinaryOp(n.Op, loadTemp, rhsTemp, n.OperandT, n.Rhs.Type().Kind)
		if err != nil {
			return err
		}
		return staged.store(p, value)

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
		if n.ErrorFormat != nil {
			errTemp := p.temp()
			p.line("const %s = %s;", errTemp, value)
			format, err := p.printIfaceCall(&ir.IfaceCall{
				Recv:      &ir.ParamRef{Name: errTemp, T: n.ErrorFormat.Recv.Type()},
				Display:   n.ErrorFormat.Display,
				MethodKey: n.ErrorFormat.MethodKey,
				Results:   n.ErrorFormat.Results,
			})
			if err != nil {
				return err
			}
			p.line("gort$.goPanicError(%s, () => %s);", errTemp, format)
			return nil
		}
		p.line("gort$.goPanicValue(%s);", value)
		return nil

	case *ir.RangeInt:
		return p.printRangeInt(n)
	}
	return fmt.Errorf("no emission for IR statement %T", stmt)
}

func (p *printer) printAssign(n *ir.AssignStmt) error {
	if n.Tuple == nil && len(n.Targets) == 1 {
		if variable, isVar := n.Targets[0].(ir.VarTarget); isVar && variable.Pkg == "" &&
			variable.T.Kind == ir.KindSlice && p.slicePlans[variable.Name] == "native-array" {
			return p.printNativeSliceAssign(variable, n.Values[0])
		}
	}
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
		ctx.returnUsed = true
		if value != "" {
			p.line("%s.v = %s;", ctx.retVar, value)
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

// labelName spells a Go label in the emitted namespace ("$" keeps it
// clear of every source identifier; JS labels live in their own space).
func labelName(label string) string { return label + "$l" }

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

// printNativeSliceAssign stores into a native-array slice local:
// self-appends push in place; the region's other admitted values
// rebind the (alias-free) name.
func (p *printer) printNativeSliceAssign(variable ir.VarTarget, value ir.Expr) error {
	if appendExpr, isAppend := value.(*ir.SliceAppend); isAppend {
		if ref, isVar := appendExpr.X.(*ir.VarRef); isVar && ref.Name == variable.Name {
			values, err := p.printArgs(appendExpr.Values)
			if err != nil {
				return err
			}
			if values != "" {
				p.line("%s.push(%s);", tsName(variable.Name), values)
			}
			return nil
		}
	}
	printed, err := p.printNativeSliceValue(variable.T, value)
	if err != nil {
		return err
	}
	p.line("%s = %s;", tsName(variable.Name), printed)
	return nil
}
