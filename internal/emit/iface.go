package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// rttiRef spells a reference to a concrete type's shared rtti object.
func (p *printer) rttiRef(r ir.RttiRef) (string, error) {
	if r.Predeclared != "" {
		return "goif$.goRtti$" + r.Predeclared, nil
	}
	if r.Composite != "" {
		externID := "undefined"
		if r.ExternID != "" {
			externID = fmt.Sprintf("%q", r.ExternID)
		}
		return fmt.Sprintf("goif$.goRttiComposite(%q, %q, %s)", r.Composite, r.Display, externID), nil
	}
	name := r.TypeName + "$rtti"
	if r.Pointer {
		name = r.TypeName + "$rttiPtr"
	}
	return p.module.symbol(r.Pkg, name)
}

// printRtti emits the shared rtti constants of one named type: the
// value-type rtti and, for struct classes, the pointer-type rtti. The
// method table maps method names onto the generated method functions;
// Go's method-set rules guarantee only reachable entries are ever
// dispatched.
func printRtti(out *strings.Builder, module *Module, typeName string, exported, pointer bool, methods []*ir.Func, promoted []ir.PromotedDelegate) error {
	p := &printer{out: out, module: module}
	sorted := append([]*ir.Func{}, methods...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	entries := make([]string, 0, len(sorted)+len(promoted))
	for _, method := range sorted {
		entries = append(entries, fmt.Sprintf("%s: %s$%s", method.Name, typeName, method.Name))
	}
	for _, delegate := range promoted {
		// A promoted method delegates through the embedded value fields
		// to the declaring type's generated function.
		target, err := p.module.symbol(delegate.Pkg, delegate.TypeName+"$"+delegate.Name)
		if err != nil {
			return err
		}
		chain := "($r as " + tsName(typeName) + ")"
		for _, field := range delegate.Path {
			chain += "." + field
		}
		entries = append(entries, fmt.Sprintf("%s: ($r: unknown, ...$a: unknown[]) => %s(%s, ...$a)", delegate.Name, target, chain))
	}
	table := "{ " + strings.Join(entries, ", ") + " }"
	if len(entries) == 0 {
		table = "{}"
	}
	export := ""
	if exported {
		export = "export "
	}
	display := module.PkgName + "." + typeName
	p.line("%sconst %s$rtti: goif$.GoRtti = { d: %q, m: %s };", export, typeName, display, table)
	if pointer {
		p.line("%sconst %s$rttiPtr: goif$.GoRtti = { d: %q, m: %s, p: true };", export, typeName, "*"+display, table)
	}
	return nil
}

// printTypeSwitch lowers a type switch onto rtti identity tests in
// clause order: the operand evaluates once, non-default clauses test in
// source order, and the default clause is the final else.
func (p *printer) printTypeSwitch(n *ir.TypeSwitchStmt) error {
	userLabel := p.takeLoopLabel()
	if n.BreakLabel != "" {
		// A direct break inside a clause exits this labeled block; a Go
		// label on the type switch nests outside it.
		p.line("%s%s: {", userLabel, labelName(n.BreakLabel))
	} else if userLabel != "" {
		p.line("%s{", userLabel)
	} else {
		p.line("{")
	}
	p.indent++
	if n.Init != nil {
		if err := p.printStmt(n.Init); err != nil {
			return err
		}
	}
	operand, err := p.printExpr(n.X)
	if err != nil {
		return err
	}
	boxTemp := p.temp()
	p.line("const %s = %s;", boxTemp, operand)

	var defaultClause *ir.TypeSwitchClause
	first := true
	for i := range n.Clauses {
		clause := &n.Clauses[i]
		if clause.Targets == nil {
			defaultClause = clause
			continue
		}
		conditions := make([]string, 0, len(clause.Targets))
		for _, target := range clause.Targets {
			if target.Nil {
				conditions = append(conditions, "("+boxTemp+" === undefined)")
				continue
			}
			rtti, err := p.rttiRef(target.Rtti)
			if err != nil {
				return err
			}
			conditions = append(conditions, "goif$.goIfaceIs("+boxTemp+", "+rtti+")")
		}
		keyword := "} else if"
		if first {
			keyword = "if"
			first = false
		}
		p.line("%s (%s) {", keyword, strings.Join(conditions, " || "))
		p.indent++
		if err := p.printTypeSwitchClause(n, clause, boxTemp); err != nil {
			return err
		}
		p.indent--
	}
	if defaultClause != nil {
		if first {
			// Only a default clause: it always runs.
			if err := p.printTypeSwitchClause(n, defaultClause, boxTemp); err != nil {
				return err
			}
		} else {
			p.line("} else {")
			p.indent++
			if err := p.printTypeSwitchClause(n, defaultClause, boxTemp); err != nil {
				return err
			}
			p.indent--
			p.line("}")
		}
	} else if !first {
		p.line("}")
	}
	p.indent--
	p.line("}")
	return nil
}

// printTypeSwitchClause binds the clause variable — the unboxed value
// for a single concrete clause, the interface value otherwise — and
// prints the body.
func (p *printer) printTypeSwitchClause(n *ir.TypeSwitchStmt, clause *ir.TypeSwitchClause, boxTemp string) error {
	if n.Bind != "" {
		spelled, err := p.tsType(clause.BindType)
		if err != nil {
			return err
		}
		if clause.BindType.Kind == ir.KindIface {
			p.line("let %s: %s = %s;", tsName(n.Bind), spelled, boxTemp)
		} else {
			value := fmt.Sprintf("((%s as goif$.GoIfaceBox).v as (%s))", boxTemp, spelled)
			if clause.BindType.Kind == ir.KindStruct {
				// The asserted struct value binds as a copy.
				value += ".goClone$()"
			}
			if clause.BindType.Kind == ir.KindArray {
				cloneElem, err := p.arrayElemClone(*clause.BindType.Elem)
				if err != nil {
					return err
				}
				value = "gosl$.goArrayClone(" + value + ", " + cloneElem + ")"
			}
			if t := clause.BindType; t.Kind == ir.KindExternal {
				value = fmt.Sprintf("(goext$.goExternalCall(%q, [%s]) as %s)",
					t.Pkg+"."+t.Named+".goClone$", value, spelled)
			}
			p.line("let %s: %s = %s;", tsName(n.Bind), spelled, value)
		}
	}
	return p.printSwitchClauseBody(clause.Body)
}
