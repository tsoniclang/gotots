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
		if r.ExternID == "" {
			fields := fmt.Sprintf("d: %q", r.Display)
			switch r.CompositeEq {
			case "uncomparable":
				fields += ", c: false"
			case "identity":
				fields += ", c: true, p: true"
			case "array-prim":
				fields += ", c: true, e: ($a: unknown, $b: unknown) => gosl$.goArrayEqual($a as unknown[], $b as unknown[])"
			}
			// "unknown" omits c: equality over it fails closed.
			return fmt.Sprintf("goif$.goRttiComposite(%q, { %s })", r.Composite, fields), nil
		}
		// An external named type's rtti is an identity token plus its
		// method-name data (for the assertion diagnostic); dispatch
		// itself is the generated exhaustive token switch that calls the
		// external contract stubs directly. Comparability stays unknown.
		names := make([]string, 0, len(p.module.ExternMethods[r.ExternID]))
		for _, method := range p.module.ExternMethods[r.ExternID] {
			names = append(names, fmt.Sprintf("%q", method.Name))
		}
		sort.Strings(names)
		return fmt.Sprintf("goif$.goRttiComposite(%q, { d: %q, ms: [%s], x: %q })", r.Composite, r.Display, strings.Join(names, ", "), r.ExternID), nil
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
// RttiInfo carries one named type's method sets and comparability for
// the value and pointer rttis.
type RttiInfo struct {
	TypeName   string
	Exported   bool
	Pointer    bool // emit the pointer rtti (struct classes and pointables)
	Comparable bool // the concrete value participates in exact equality
	HasEq      bool // a goEq$ method is generated for the value
	Methods    []*ir.Func
	Promoted   []ir.PromotedDelegate
}

// printRtti emits one named type's rttis. Go's method sets are exact:
// the VALUE rtti carries only value-receiver methods (and value-receiver
// promotions); the POINTER rtti carries every method. Tables are keyed
// by canonical dispatch identity — name-only collisions can never
// satisfy a method-set test or dispatch. Comparable concrete types
// carry a comparability flag and a goEq$ reference so interface equality
// is exact and uncomparable dynamic types panic.
func printRtti(out *strings.Builder, module *Module, info RttiInfo) error {
	p := &printer{out: out, module: module}
	export := "export "
	display := module.PkgName + "." + info.TypeName
	eqSuffix := ""
	if info.HasEq {
		eqSuffix = fmt.Sprintf(", e: ($a: unknown, $b: unknown) => ($a as %s).goEq$($b as %s)", tsName(info.TypeName), tsName(info.TypeName))
	}
	comparable := "false"
	if info.Comparable {
		comparable = "true"
	}
	valueMethods, pointerMethods := info.methodDisplays()
	p.line("%sconst %s$rtti: goif$.GoRtti = { d: %q, c: %s, ms: %s%s };", export, info.TypeName, display, comparable, valueMethods, eqSuffix)
	if info.Pointer {
		// A pointer's dynamic type is always comparable (identity) and
		// has no goEq$ of its own.
		p.line("%sconst %s$rttiPtr: goif$.GoRtti = { d: %q, c: true, p: true, ms: %s };", export, info.TypeName, "*"+display, pointerMethods)
	}
	return nil
}

// methodDisplays spells the sorted method-name lists of the value and
// pointer method sets — data used only for the missing-method
// diagnostic of a failed interface assertion, never for dispatch.
func (info RttiInfo) methodDisplays() (string, string) {
	valueSet := map[string]bool{}
	pointerSet := map[string]bool{}
	for _, method := range info.Methods {
		pointerSet[method.Name] = true
		if !method.PointerReceiver {
			valueSet[method.Name] = true
		}
	}
	for _, delegate := range info.Promoted {
		pointerSet[delegate.Name] = true
		if delegate.ValueReceiver {
			valueSet[delegate.Name] = true
		}
	}
	return methodList(valueSet), methodList(pointerSet)
}

func methodList(set map[string]bool) string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
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
				callee, err := p.module.symbol(t.Pkg, externCloneSymbol(t.Named))
				if err != nil {
					return err
				}
				value = fmt.Sprintf("%s(%s)", callee, value)
			}
			p.line("let %s: %s = %s;", tsName(n.Bind), spelled, value)
		}
	}
	return p.printSwitchClauseBody(clause.Body)
}
