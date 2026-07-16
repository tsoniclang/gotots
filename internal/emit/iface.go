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
			fields := fmt.Sprintf("d: %q, m: {}", r.Display)
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
		// An external named type's rtti carries a STATIC method table over
		// the unit-recorded contract stubs (keyed by canonical dispatch
		// identity); anything beyond the recorded surface fails closed at
		// the dispatch site, and its comparability stays unknown.
		dot := strings.LastIndex(r.ExternID, ".")
		externPkg, typeName := r.ExternID[:dot], r.ExternID[dot+1:]
		entries := make([]string, 0, len(p.module.ExternMethods[r.ExternID]))
		for _, method := range p.module.ExternMethods[r.ExternID] {
			callee, err := p.module.symbol(externPkg, externMethodSymbol(typeName, method.Name))
			if err != nil {
				return "", err
			}
			entries = append(entries, fmt.Sprintf("%q: %s", method.Key, callee))
		}
		return fmt.Sprintf("goif$.goRttiComposite(%q, { d: %q, m: { %s }, x: %q })", r.Composite, r.Display, strings.Join(entries, ", "), r.ExternID), nil
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
	valueEntry, pointerEntry, err := p.rttiTables(info)
	if err != nil {
		return err
	}
	export := ""
	if info.Exported {
		export = "export "
	}
	display := module.PkgName + "." + info.TypeName
	eqSuffix := ""
	if info.HasEq {
		eqSuffix = fmt.Sprintf(", e: ($a: unknown, $b: unknown) => ($a as %s).goEq$($b as %s)", tsName(info.TypeName), tsName(info.TypeName))
	}
	comparable := "false"
	if info.Comparable {
		comparable = "true"
	}
	p.line("%sconst %s$rtti: goif$.GoRtti = { d: %q, c: %s, m: %s%s };", export, info.TypeName, display, comparable, valueEntry, eqSuffix)
	if info.Pointer {
		// A pointer's dynamic type is always comparable (identity) and
		// has no goEq$ of its own.
		p.line("%sconst %s$rttiPtr: goif$.GoRtti = { d: %q, c: true, p: true, m: %s };", export, info.TypeName, "*"+display, pointerEntry)
	}
	return nil
}

// rttiTables builds the value and pointer method tables keyed by
// canonical dispatch identity.
func (p *printer) rttiTables(info RttiInfo) (string, string, error) {
	type entry struct{ key, spelling string }
	var valueSet, pointerSet []entry
	add := func(pointerOnly bool, key, spelling string) {
		pointerSet = append(pointerSet, entry{key, spelling})
		if !pointerOnly {
			valueSet = append(valueSet, entry{key, spelling})
		}
	}
	methods := append([]*ir.Func{}, info.Methods...)
	sort.Slice(methods, func(i, j int) bool { return methods[i].DispatchKey < methods[j].DispatchKey })
	for _, method := range methods {
		add(method.PointerReceiver, method.DispatchKey, fmt.Sprintf("%s$%s", info.TypeName, method.Name))
	}
	promoted := append([]ir.PromotedDelegate{}, info.Promoted...)
	sort.Slice(promoted, func(i, j int) bool { return promoted[i].DispatchKey < promoted[j].DispatchKey })
	for _, delegate := range promoted {
		target, err := p.module.symbol(delegate.Pkg, delegate.TypeName+"$"+delegate.Name)
		if err != nil {
			return "", "", err
		}
		chain := "($r as " + tsName(info.TypeName) + ")"
		for _, field := range delegate.Path {
			chain += "." + field
		}
		spelling := fmt.Sprintf("($r: unknown, ...$a: unknown[]) => %s(%s, ...$a)", target, chain)
		add(!delegate.ValueReceiver, delegate.DispatchKey, spelling)
	}
	spell := func(entries []entry) string {
		if len(entries) == 0 {
			return "{}"
		}
		parts := make([]string, len(entries))
		for i, e := range entries {
			parts[i] = fmt.Sprintf("%q: %s", e.key, e.spelling)
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	}
	return spell(valueSet), spell(pointerSet), nil
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
