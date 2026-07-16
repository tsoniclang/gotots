package emit

import (
	"crypto/sha256"
	"encoding/hex"
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
			if target.Rtti.Pkg != "" && p.module.Withheld != nil && p.module.Withheld(target.Rtti.Pkg) {
				// A withheld package's token cannot exist at runtime; the
				// case can never match and must not reference the token.
				continue
			}
			// Literal-discriminant comparison NARROWS the union member,
			// so the clause binds the exact payload with no cast.
			conditions = append(conditions, fmt.Sprintf("(%s !== undefined && %s.k === %q)", boxTemp, boxTemp, boxDiscriminant(target.Rtti)))
		}
		if len(conditions) == 0 {
			// Every target withheld: the clause is unreachable.
			continue
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
			// The clause guard narrowed boxTemp to the member: .v is the
			// exact payload — no recovery cast.
			value := boxTemp + ".v"
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

// boxDiscriminant spells the box's literal token for one rtti reference:
// the canonical dynamic-type identity the union discriminates on.
func boxDiscriminant(r ir.RttiRef) string {
	switch {
	case r.Predeclared != "":
		// Disjoint namespace: a predeclared literal can never fall into
		// the composite template member, so literal narrowing is exact.
		return "p:" + r.Predeclared
	case r.ExternID != "":
		// External NAMED types are union members with canonical ids —
		// never the open-composite namespace.
		return r.Composite
	case r.Composite != "":
		return "c:" + r.Composite
	case r.Pointer:
		return "*" + r.Pkg + "." + r.TypeName
	default:
		return r.Pkg + "." + r.TypeName
	}
}

// predeclaredMembers spells the fifteen predeclared union members with
// their exact payload carriers.
var predeclaredMembers = []struct{ name, payload string }{
	{"bool", "boolean"}, {"string", "string"},
	{"int", "goabi$.GoInt"}, {"int8", "number"}, {"int16", "number"}, {"int32", "number"},
	{"int64", "goabi$.GoInt64"}, {"uint", "goabi$.GoUint"}, {"uint8", "number"},
	{"uint16", "number"}, {"uint32", "number"}, {"uint64", "goabi$.GoUint64"},
	{"uintptr", "goabi$.GoUintptr"}, {"float32", "number"}, {"float64", "number"},
}

// boxVtable spells the box's vtable operand: the concrete type's shared
// const for owned named types (flavor-exact), an inline typed adapter
// object over stub exports for external named types, and the empty
// object for methodless predeclared and composite types.
func (p *printer) boxVtable(r ir.RttiRef) (string, error) {
	switch {
	case r.Predeclared != "" || (r.Composite != "" && r.ExternID == ""):
		return "{}", nil
	case r.ExternID != "":
		methods := p.module.ExternMethods[r.ExternID]
		if len(methods) == 0 {
			return "{}", nil
		}
		entries := make([]string, 0, len(methods))
		for _, method := range methods {
			if method.Adapter == "" {
				continue
			}
			entries = append(entries, method.Name+": "+method.Adapter)
		}
		return "{ " + joinComma(entries) + " }", nil
	case r.Pointer:
		return p.module.symbol(r.Pkg, r.TypeName+"$vtablePtr")
	default:
		return p.module.symbol(r.Pkg, r.TypeName+"$vtable")
	}
}

// TypedAdapter spells one exactly typed vtable arrow: the receiver and
// parameters at their declared types, delegating to callee.
func TypedAdapter(module *Module, params []ir.Var, results []ir.Type, callee string) (string, error) {
	p := &printer{module: module}
	parts := make([]string, 0, len(params))
	names := make([]string, 0, len(params))
	for i, param := range params {
		spelled, err := p.tsType(param.Type)
		if err != nil {
			return "", err
		}
		name := fmt.Sprintf("$a%d", i)
		parts = append(parts, name+": "+spelled)
		names = append(names, name)
	}
	result, err := p.tsFuncResultType(results)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s): %s => %s(%s)", joinComma(parts), result, callee, joinComma(names)), nil
}

// TypedAdapterType spells the arrow TYPE of a typed adapter.
func TypedAdapterType(module *Module, params []ir.Var, results []ir.Type) (string, error) {
	p := &printer{module: module}
	parts := make([]string, 0, len(params))
	for i, param := range params {
		spelled, err := p.tsType(param.Type)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("$a%d: %s", i, spelled))
	}
	result, err := p.tsFuncResultType(results)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s) => %s", joinComma(parts), result), nil
}

// ifaceUnionAlias spells one interface type as its closed discriminated
// union alias (ADR-0004): one GoBox member per implementer with exact
// literal, payload, and vtable types, undefined for nil, and — for the
// empty interface — the predeclared and composite members. Payload and
// vtable references use type-only symbols, so alias spelling adds no
// runtime import edge.
func (p *printer) ifaceUnionAlias(t ir.Type) (string, error) {
	identity := t.IfaceID
	if identity == "" {
		identity = t.Go
	}
	digest := sha256.Sum256([]byte(identity))
	name := "Iface$" + hex.EncodeToString(digest[:6])
	if p.module == nil {
		return name, nil
	}
	if _, exists := p.module.ifaceAliases[name]; exists {
		return name, nil
	}
	// Reserve first: member payloads may mention this same interface
	// (via method signatures they do not — payloads are concrete — but
	// reservation keeps registration idempotent under recursion).
	p.module.RegisterIfaceAlias(name, "")
	members := []string{"undefined"}
	for _, member := range p.retainedMembers(t) {
		payload, err := p.memberPayload(member)
		if err != nil {
			return "", err
		}
		var vtable string
		if member.Extern {
			// External implementers carry inline stub-adapter vtables;
			// the member type is the exact structural adapter surface.
			entries := []string{}
			for _, method := range p.module.ExternMethods[member.Pkg+"."+member.Type] {
				if method.AdapterType != "" {
					entries = append(entries, method.Name+": "+method.AdapterType)
				}
			}
			vtable = "{ " + joinComma(entries) + " }"
			if len(entries) == 0 {
				vtable = "Record<never, never>"
			}
		} else {
			suffix := "$vtable"
			if member.Pointer {
				suffix = "$vtablePtr"
			}
			ref, err := p.module.typeSymbol(member.Pkg, member.Type+suffix)
			if err != nil {
				return "", err
			}
			vtable = "typeof " + ref
		}
		members = append(members, fmt.Sprintf("goif$.GoBox<%q, %s, %s>", member.K, payload, vtable))
	}
	if t.IfaceEmpty {
		// The empty interface accepts every predeclared type (exact
		// payload members) and every composite type (one template-literal
		// member in the disjoint "c:" namespace, whose payload re-emerges
		// only through token-checked assertions — ADR-0004).
		for _, member := range predeclaredMembers {
			members = append(members, fmt.Sprintf("goif$.GoBox<%q, %s, Record<never, never>>", "p:"+member.name, member.payload))
		}
		members = append(members, "goif$.GoCompositeBox")
	}
	declaration := "type " + name + " = " + strings.Join(members, " | ") + ";"
	p.module.ifaceAliases[name] = declaration
	return name, nil
}

// memberPayload spells one union member's payload by name: class
// instances are identity carriers (the pointer IS the instance, nilable
// when boxed through a pointer); named value carriers box the value or
// its cell; external handles are branded and nilable through pointers.
func (p *printer) memberPayload(member ir.IfaceMember) (string, error) {
	if member.Extern {
		handle := fmt.Sprintf("goext$.GoExtern<%q>", member.Pkg+"."+member.Type)
		if member.Pointer {
			return "(" + handle + " | undefined)", nil
		}
		return handle, nil
	}
	base, err := p.module.typeSymbol(member.Pkg, member.Type)
	if err != nil {
		return "", err
	}
	if !member.Pointer {
		return base, nil
	}
	if member.Struct {
		return "(" + base + " | undefined)", nil
	}
	return "(gort$.GoCell<" + base + "> | undefined)", nil
}

// retainedMembers filters an interface's implementer union to the
// packages retained in this bundle: a withheld package's class exists
// in no runnable module, so nothing of that type can box at runtime and
// no alias or dispatch branch may reference it.
func (p *printer) retainedMembers(t ir.Type) []ir.IfaceMember {
	if p.module == nil || p.module.Withheld == nil {
		return t.IfaceMembers
	}
	out := make([]ir.IfaceMember, 0, len(t.IfaceMembers))
	for _, member := range t.IfaceMembers {
		if !member.Extern && p.module.Withheld(member.Pkg) {
			continue
		}
		out = append(out, member)
	}
	return out
}
