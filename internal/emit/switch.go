// Switch emission: the tag evaluates once, clauses match in source
// order by the tag carrier's exact equality, and fallthrough transfers
// unconditionally.
package emit

import (
	"github.com/tsoniclang/gotots/internal/ir"
)

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
	// The tag switches under its full declared carrier type: a folded
	// constant tag must not narrow to its literal type, which would make
	// non-matching literal cases a type error.
	spelledTag, err := p.tsType(n.Tag.Type())
	if err != nil {
		return err
	}
	p.line("%sswitch (%s as (%s)) {", p.takeLoopLabel(), tag, spelledTag)
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
