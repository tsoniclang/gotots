package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// requireIdentity is the single choke point every method-slot / method-key
// emission passes through: a canonical dispatch identity is populated by
// the IR builder (which fails a declaration closed on any MethodKey/slot
// failure), so an empty value here is a construction-invariant violation
// and panics at generation — it can NEVER silently become a bare-name
// substitution.
func requireIdentity(value, what string) string {
	if value == "" {
		panic("emit: " + what + " has no canonical identity")
	}
	return value
}

// rttiRef spells a reference to a concrete type's shared rtti object.
func (p *printer) rttiRef(r ir.RttiRef) (string, error) {
	if r.Predeclared != "" {
		return "goif$.goRtti$" + r.Predeclared, nil
	}
	if r.Composite != "" {
		if r.ExternID == "" {
			// The rtti is a pure identity+display token; interface equality
			// is generated inline per exact union member, never through the
			// rtti, so it carries no comparability or equality function. A
			// pointer composite keeps p (identity) for diagnostics parity.
			fields := fmt.Sprintf("d: %q", r.Display)
			if r.CompositeEq == "identity" {
				fields += ", p: true"
			}
			return fmt.Sprintf("goif$.goRttiComposite(%q, { %s })", r.Composite, fields), nil
		}
		// An external named type's rtti is an identity token plus its
		// method-name data (for the assertion diagnostic); dispatch
		// itself is the generated exhaustive token switch that calls the
		// external contract stubs directly. Comparability stays unknown.
		names := make([]string, 0, len(p.module.ExternMethods[r.ExternID]))
		for _, method := range p.module.ExternMethods[r.ExternID] {
			names = append(names, fmt.Sprintf("%q", requireIdentity(method.Key, "external method "+method.Name)))
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
	// The rtti is a pure identity+display+method-name token: interface
	// equality is generated inline per exact union member, so the rtti
	// carries no comparability flag and no equality function (no erasure).
	valueMethods, pointerMethods := info.methodDisplays()
	p.line("%sconst %s$rtti: goif$.GoRtti = { d: %q, ms: %s };", export, info.TypeName, display, valueMethods)
	if info.Pointer {
		p.line("%sconst %s$rttiPtr: goif$.GoRtti = { d: %q, p: true, ms: %s };", export, info.TypeName, "*"+display, pointerMethods)
	}
	return nil
}

// methodDisplays spells the sorted method-IDENTITY lists of the value and
// pointer method sets — each entry the method's canonical MethodKey (name,
// unexported package, signature digest) — data used only for the
// signature-aware missing-method diagnostic of a failed interface
// assertion, never for dispatch. Every identity is populated by the IR
// builder (MethodKey is total); an empty one fails closed through
// requireIdentity, never a bare-name fallback.
func (info RttiInfo) methodDisplays() (string, string) {
	valueSet := map[string]bool{}
	pointerSet := map[string]bool{}
	// The identity is the canonical MethodKey, always populated by the IR
	// builder (it is total). An empty MethodIdent is a construction defect,
	// not a bare-name fallback — a method with no canonical identity would
	// have failed the declaration closed already.
	for _, method := range info.Methods {
		id := requireIdentity(method.MethodIdent, "method "+method.Name)
		pointerSet[id] = true
		if !method.PointerReceiver {
			valueSet[id] = true
		}
	}
	for _, delegate := range info.Promoted {
		id := requireIdentity(delegate.MethodIdent, "promoted method "+delegate.Name)
		pointerSet[id] = true
		if delegate.ValueReceiver {
			valueSet[id] = true
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
	if clause.Bind != "" {
		spelled, err := p.tsType(clause.BindType)
		if err != nil {
			return err
		}
		if clause.BindType.Kind == ir.KindIface {
			p.line("let %s: %s = %s;", tsName(clause.Bind), spelled, boxTemp)
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
			p.line("let %s: %s = %s;", tsName(clause.Bind), spelled, value)
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
	case r.InstType != nil:
		// An instantiated-generic box: the inline vtable over the
		// generated generic functions with the instantiation's exact
		// factory derivations.
		return p.instVtableValue(*r.InstType, r.InstSlots, r.Pointer)
	case r.Predeclared != "" || (r.Composite != "" && r.ExternID == ""):
		return "{}", nil
	case r.ExternID != "":
		methods := p.module.ExternMethods[r.ExternID]
		if len(methods) == 0 {
			return "{}", nil
		}
		entries := make([]string, 0, len(methods))
		for _, method := range methods {
			adapter := method.Adapter
			if r.Pointer {
				adapter = method.AdapterPtr
			}
			if adapter == "" {
				continue
			}
			slot := requireIdentity(method.Slot, "external vtable slot for "+method.Name)
			entries = append(entries, slot+": "+adapter)
		}
		return "{ " + joinComma(entries) + " }", nil
	case r.Pointer:
		return p.module.symbol(r.Pkg, r.TypeName+"$vtablePtr")
	default:
		return p.module.symbol(r.Pkg, r.TypeName+"$vtable")
	}
}

// ExternAdapterForms carries one external method's two exactly typed
// vtable arrows — the VALUE-member form and the POINTER-member form —
// with their arrow types. The receiver's representation IS the union
// member's boxed payload (the single rule memberPayload spells): a
// basic-underlying external type boxes its value carrier and a pointer
// member carries the value's cell, dereferenced with Go's nil panic
// before the delegated call; every other external type boxes its branded
// handle, nilable through a pointer.
type ExternAdapterForms struct {
	Adapter        string
	AdapterType    string
	AdapterPtr     string
	AdapterPtrType string
}

// ExternAdapters builds both member forms of one external method's vtable
// arrow. valueRecv is the boxed VALUE payload type (the basic carrier, or
// the branded handle); basicCarrier selects the cell-deref pointer form.
// params excludes the receiver; callee is the stub export.
func ExternAdapters(module *Module, valueRecv ir.Type, basicCarrier bool, params []ir.Var, results []ir.Type, callee string) (ExternAdapterForms, error) {
	forms := ExternAdapterForms{}
	valueParams := append([]ir.Var{{Name: "recv", Type: valueRecv}}, params...)
	var err error
	if forms.Adapter, err = TypedAdapter(module, valueParams, results, callee); err != nil {
		return forms, err
	}
	if forms.AdapterType, err = TypedAdapterType(module, valueParams, results); err != nil {
		return forms, err
	}
	pointerRecv := ir.Type{Kind: ir.KindPointer, Go: "*" + valueRecv.Go, Named: valueRecv.Named, Pkg: valueRecv.Pkg, Elem: &valueRecv}
	pointerParams := append([]ir.Var{{Name: "recv", Type: pointerRecv}}, params...)
	if forms.AdapterPtrType, err = TypedAdapterType(module, pointerParams, results); err != nil {
		return forms, err
	}
	if !basicCarrier {
		// The handle is the value: the pointer member passes it through
		// (nil reaches the implementation's exact nil semantics).
		if forms.AdapterPtr, err = TypedAdapter(module, pointerParams, results, callee); err != nil {
			return forms, err
		}
		return forms, nil
	}
	// A pointer member carries the value's cell: Go's implicit deref for a
	// value-receiver call panics on nil, then the held value delegates.
	forms.AdapterPtr, err = typedAdapterDeref(module, pointerParams, results, callee)
	return forms, err
}

// typedAdapterDeref spells the pointer-member arrow of a basic-carrier
// external method: the cell receiver nil-checks and dereferences before
// the delegated call.
func typedAdapterDeref(module *Module, params []ir.Var, results []ir.Type, callee string) (string, error) {
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
		if i == 0 {
			checked, err := p.nilCheckOf(name, param.Type)
			if err != nil {
				return "", err
			}
			name = checked + ".v"
		}
		names = append(names, name)
	}
	result, err := p.tsFuncResultType(results)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s): %s => %s(%s)", joinComma(parts), result, callee, joinComma(names)), nil
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

// ifaceAliasName is the union-alias name for one canonical interface
// identity: the FULL digest of the identity, so distinct interfaces can
// never collide onto one alias (a truncated prefix is not an injective
// identity, and the alias name is an internal TS identifier so its length
// is free). Both the alias declaration and every reference to a union's
// equality function derive the name through this one function, so they
// always agree.

// instSlotRecv spells one instantiated slot's receiver parameter type —
// the MEMBER's payload form (a pointer member's payload is nilable
// identity; a value member's payload is the instance).
func (p *printer) instSlotRecv(instType ir.Type, memberPointer bool) (string, error) {
	instSpelled, err := p.tsType(instType)
	if err != nil {
		return "", err
	}
	if memberPointer {
		return instSpelled + " | undefined", nil
	}
	return instSpelled, nil
}

// instSlotResult spells one instantiated slot's result type.
func (p *printer) instSlotResult(slot ir.InstSlot) (string, error) {
	switch len(slot.Results) {
	case 0:
		return "void", nil
	case 1:
		return p.tsType(slot.Results[0])
	}
	parts := make([]string, len(slot.Results))
	for i, result := range slot.Results {
		spelled, err := p.tsType(result)
		if err != nil {
			return "", err
		}
		parts[i] = spelled
	}
	return "readonly [" + joinComma(parts) + "]", nil
}

// instVtableValue spells the inline vtable OBJECT of one instantiated
// generic: each slot is an exactly typed arrow dispatching to the
// generated generic function with the instantiation's factory
// derivations (the same derivations every direct call site passes).
func (p *printer) instVtableValue(instType ir.Type, slots []ir.InstSlot, memberPointer bool) (string, error) {
	instSpelled, err := p.tsType(instType)
	if err != nil {
		return "", err
	}
	entries := make([]string, 0, len(slots))
	for _, slot := range slots {
		recvSpelled, err := p.instSlotRecv(instType, memberPointer)
		if err != nil {
			return "", err
		}
		params := []string{"$r: " + recvSpelled}
		// A VALUE-receiver method through a POINTER member derefs (nil
		// panics, exactly Go); every other combination passes through.
		recvArg := "$r"
		if memberPointer && !slot.PointerRecv {
			recvArg = "gort$.goNilCheck<" + instSpelled + ">($r)"
		}
		args := []string{recvArg}
		for i, param := range slot.Params {
			spelled, err := p.tsType(param)
			if err != nil {
				return "", err
			}
			params = append(params, fmt.Sprintf("$a%d: %s", i, spelled))
			args = append(args, fmt.Sprintf("$a%d", i))
		}
		result, err := p.instSlotResult(slot)
		if err != nil {
			return "", err
		}
		vtClass := instType.Named
		if instType.MapFamilyEnc {
			vtClass += "$ek"
		}
		callee, err := p.module.symbol(instType.Pkg, vtClass+"$"+slot.MethodName)
		if err != nil {
			return "", err
		}
		typeArgs := make([]string, len(slot.TypeArgs))
		for i, arg := range slot.TypeArgs {
			if typeArgs[i], err = p.tsType(arg); err != nil {
				return "", err
			}
		}
		factories, err := p.zeroFactoryArgs(slot.TypeArgs, slot.KeyedParams)
		if err != nil {
			return "", err
		}
		rttiSlots, err := p.rttiFactoryArgs(slot.RttiParams, slot.RttiArgs, nil)
		if err != nil {
			return "", err
		}
		if rttiSlots != "" {
			factories += ", " + rttiSlots
		}
		call := callee + "<" + joinComma(typeArgs) + ">(" + joinComma(args)
		if factories != "" {
			call += ", " + factories
		}
		call += ")"
		entries = append(entries, fmt.Sprintf("%s: (%s): %s => %s",
			requireIdentity(slot.Slot, "instantiated vtable slot for "+slot.MethodName), strings.Join(params, ", "), result, call))
	}
	return "{ " + joinComma(entries) + " }", nil
}

// instVtableType spells the inline vtable TYPE of one instantiated
// generic member (the union member's m surface).
func (p *printer) instVtableType(instType ir.Type, slots []ir.InstSlot, memberPointer bool) (string, error) {
	entries := make([]string, 0, len(slots))
	for _, slot := range slots {
		recvSpelled, err := p.instSlotRecv(instType, memberPointer)
		if err != nil {
			return "", err
		}
		params := []string{"$r: " + recvSpelled}
		for i, param := range slot.Params {
			spelled, err := p.tsType(param)
			if err != nil {
				return "", err
			}
			params = append(params, fmt.Sprintf("$a%d: %s", i, spelled))
		}
		result, err := p.instSlotResult(slot)
		if err != nil {
			return "", err
		}
		entries = append(entries, fmt.Sprintf("%s: (%s) => %s",
			requireIdentity(slot.Slot, "instantiated vtable slot for "+slot.MethodName), strings.Join(params, ", "), result))
	}
	if len(entries) == 0 {
		return "Record<never, never>", nil
	}
	return "{ " + joinComma(entries) + " }", nil
}
