// Declaration-statement emission: plain, tuple-bound, boxed, and
// planner-selected native-slice locals, plus the loop/switch body
// helpers that scope branch transforms.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

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
				callee, err := p.module.symbol(t.Pkg, externCloneSymbol(t.Named))
				if err != nil {
					return err
				}
				p.line("let %s: %s = %s(%s[%d]);", tsName(name), spelled, callee, tuple, i)
				continue
			}
			p.line("let %s: %s = %s[%d];", tsName(name), spelled, tuple, i)
		}
		return nil
	}
	for i, name := range n.Names {
		if name != "_" && n.Types[i].Kind == ir.KindSlice && p.slicePlans[name] == "native-array" {
			// The planner selected the native array for this region: the
			// declaration takes the direct form, and every value the
			// region admits (literal, make, self-append, zero) lowers
			// directly.
			if err := p.printNativeSliceDecl(name, n.Types[i], n.Values[i]); err != nil {
				return err
			}
			continue
		}
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

// printNativeSliceDecl declares one native-array slice local.
func (p *printer) printNativeSliceDecl(name string, t ir.Type, value ir.Expr) error {
	element, err := p.tsType(*t.Elem)
	if err != nil {
		return err
	}
	printed, err := p.printNativeSliceValue(t, value)
	if err != nil {
		return err
	}
	p.line("let %s: %s[] = %s;", tsName(name), element, printed)
	return nil
}

// printNativeSliceValue spells a native-region slice value: literals
// and makes construct plain arrays; nil zeros are empty arrays (the
// region never observes nil).
func (p *printer) printNativeSliceValue(t ir.Type, value ir.Expr) (string, error) {
	switch n := value.(type) {
	case *ir.SliceLit:
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "[" + values + "]", nil
	case *ir.SliceMake:
		length, err := p.printExpr(n.Length)
		if err != nil {
			return "", err
		}
		zero, err := p.zeroLiteral(*t.Elem)
		if err != nil {
			return "", err
		}
		make := "gosl$.goArrayZero(Number(" + length + "), () => " + zero + ")"
		if n.Capacity != nil {
			capacity, err := p.printExpr(n.Capacity)
			if err != nil {
				return "", err
			}
			// The capacity hint evaluates (the region never observes it).
			make = "(void (" + capacity + "), " + make + ")"
		}
		return make, nil
	case *ir.NilConst:
		// The region never compares against nil, so its zero is the
		// empty backing.
		return "[]", nil
	case *ir.VarRef:
		return p.printExpr(n)
	}
	return "", fmt.Errorf("native slice region admitted an unplanned value %T", value)
}
