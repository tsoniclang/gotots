// Per-type dispatch vtables (ADR-0004): one shared const per method-set
// flavor with exactly typed adapters, generated in the type's own
// package so no dispatch site ever imports an implementer.
package emit

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/plan"
)

// printVtables emits the type's shared dispatch tables: one const per
// method-set flavor, each entry an exactly typed adapter to the
// generated method function. A value-receiver method reached through
// the pointer flavor dereferences with Go's nil panic; promoted methods
// chain through embedded fields (pointer steps nil-check) — all inside
// the adapter, in the type's own package, so no dispatch site ever
// imports an implementer.
func printVtables(out *strings.Builder, module *Module, info RttiInfo) error {
	p := &printer{out: out, module: module}
	value, pointer, err := p.vtableEntries(info)
	if err != nil {
		return err
	}
	export := "export "
	p.line("%sconst %s$vtable = { %s } as const;", export, info.TypeName, joinComma(value))
	if info.Pointer {
		p.line("%sconst %s$vtablePtr = { %s } as const;", export, info.TypeName, joinComma(pointer))
	}
	return nil
}

// vtableEntries spells the adapters of both flavors.
func (p *printer) vtableEntries(info RttiInfo) ([]string, []string, error) {
	self := tsName(info.TypeName)
	// Object-model families (ADR-0012): a boxed adapter calls a self-interface
	// CONTRACT method as a virtual member; every other family method is an
	// ordinary Go method reached through its static free function — exactly
	// the disposition ordinary dispatch uses, so there is one representation.
	familyType := ir.Type{Kind: ir.KindStruct, Named: info.TypeName, Pkg: p.module.Pkg}
	isFam := p.module.isFamilyClass(info.TypeName)
	// memberOf: "" spells the legacy free-function adapter; a member
	// name spells the ordinary class-member call on the chained
	// receiver (the ADR-0006 form).
	adapter := func(methodName string, params []ir.Var, resultVars []ir.Var, callee string, recvSpelling string, chain string, memberOf string) (string, error) {
		results := make([]ir.Type, len(resultVars))
		for i, r := range resultVars {
			results[i] = r.Type
		}
		parts := []string{"$r: " + recvSpelling}
		names := []string{}
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
		if memberOf != "" {
			return fmt.Sprintf("%s: (%s): %s => %s.%s(%s)", methodName, joinComma(parts), result, chain, memberOf, joinComma(names)), nil
		}
		operands := append([]string{chain}, names...)
		return fmt.Sprintf("%s: (%s): %s => %s(%s)", methodName, joinComma(parts), result, callee, joinComma(operands)), nil
	}
	var valueSet, pointerSet []string
	methods := append([]*ir.Func{}, info.Methods...)
	sort.Slice(methods, func(i, j int) bool { return methods[i].Name < methods[j].Name })
	for _, method := range methods {
		// The vtable PROPERTY is the method's canonical slot (never the bare
		// name); the CALLEE symbol is the generated method function. An empty
		// slot is a construction defect, not a spelling fallback.
		slot := requireIdentity(method.Slot, "vtable slot for method "+method.Name)
		callee := info.TypeName + "$" + method.Name
		memberOf := ""
		if isFam {
			if p.module.isFamilyContractMethod(familyType, method.Name) {
				memberOf = tsName(method.Name)
			}
		} else if p.module.MethodEmissionFor(methodPlanKey(method)) == plan.MethodOrdinaryNilChecked {
			memberOf = tsName(method.Name)
		}
		recvType := method.Receiver.Type
		if !method.PointerReceiver {
			// Value flavor: the payload is the receiver value exactly.
			recvSpelled, err := p.tsType(recvType)
			if err != nil {
				return nil, nil, err
			}
			entry, err := adapter(slot, method.Params, method.Results, callee, recvSpelled, "$r", memberOf)
			if err != nil {
				return nil, nil, err
			}
			valueSet = append(valueSet, entry)
			// Pointer flavor of a value-receiver method: the payload is
			// the pointer carrier; deref with Go's nil panic (identity
			// carriers ARE their pointer, cells read .v), then the method
			// itself copies on entry.
			pointerType := ir.Type{Kind: ir.KindPointer, Go: "*" + recvType.Go, Elem: &recvType}
			ptrSpelled, err := p.tsType(pointerType)
			if err != nil {
				return nil, nil, err
			}
			chain := "gort$.goNilCheck<" + recvSpelled + ">($r)"
			switch recvType.Kind {
			case ir.KindStruct, ir.KindArray, ir.KindExternal:
				// identity carrier: the deref is the instance
			default:
				cell := "gort$.GoCell<" + recvSpelled + ">"
				chain = "gort$.goNilCheck<" + cell + ">($r).v"
			}
			entryPtr, err := adapter(slot, method.Params, method.Results, callee, ptrSpelled, chain, memberOf)
			if err != nil {
				return nil, nil, err
			}
			pointerSet = append(pointerSet, entryPtr)
			continue
		}
		// Pointer receiver: the generated function takes the pointer
		// carrier itself; Go runs it on nil (body derefs panic). The
		// MEMBER form guards at the call — the ADR-0006-proven
		// equivalent of that first in-body dereference.
		recvSpelled, err := p.tsType(recvType)
		if err != nil {
			return nil, nil, err
		}
		chain := "$r"
		if memberOf != "" {
			chain = "gort$.goNilCheck($r)"
		}
		entry, err := adapter(slot, method.Params, method.Results, callee, recvSpelled, chain, memberOf)
		if err != nil {
			return nil, nil, err
		}
		pointerSet = append(pointerSet, entry)
	}
	promoted := append([]ir.PromotedDelegate{}, info.Promoted...)
	sort.Slice(promoted, func(i, j int) bool { return promoted[i].Name < promoted[j].Name })
	for _, delegate := range promoted {
		targetType := delegate.TypeName
		targetPkg := delegate.Pkg
		if delegate.IfaceField {
			// The delegate is a generated function ON THE EMBEDDING type
			// (its body dispatches the embedded interface's closed union),
			// taking the embedding receiver directly.
			targetType = info.TypeName
			targetPkg = p.module.Pkg
		}
		target, err := p.module.symbol(targetPkg, targetType+"$"+delegate.Name)
		if err != nil {
			return nil, nil, err
		}
		if delegate.IfaceField {
			slot := requireIdentity(delegate.Slot, "vtable slot for promoted method "+delegate.Name)
			valueEntry := fmt.Sprintf("%s: ($r: %s, ...$a: goif$.DropFirst<Parameters<typeof %s>>) => %s($r, ...$a)",
				slot, self, target, target)
			pointerEntry := fmt.Sprintf("%s: ($r: (%s | undefined), ...$a: goif$.DropFirst<Parameters<typeof %s>>) => %s(gort$.goNilCheck<%s>($r), ...$a)",
				slot, self, target, target, self)
			if delegate.ValueReceiver {
				valueSet = append(valueSet, valueEntry)
			}
			pointerSet = append(pointerSet, pointerEntry)
			continue
		}
		if isFam {
			slot := requireIdentity(delegate.Slot, "vtable slot for promoted method "+delegate.Name)
			// A promoted self-interface CONTRACT method is virtual — inherited
			// natively — so it dispatches through the member on $r.
			if p.module.isFamilyContractMethod(familyType, delegate.Name) {
				member := tsName(delegate.Name)
				valueEntry := fmt.Sprintf("%s: ($r: %s, ...$a: Parameters<%s[%q]>) => $r.%s(...$a)",
					slot, self, self, member, member)
				pointerEntry := fmt.Sprintf("%s: ($r: (%s | undefined), ...$a: Parameters<%s[%q]>) => gort$.goNilCheck<%s>($r).%s(...$a)",
					slot, self, self, member, self, member)
				if delegate.ValueReceiver {
					valueSet = append(valueSet, valueEntry)
				}
				pointerSet = append(pointerSet, pointerEntry)
				continue
			}
			// An ordinary method promoted through the PRIMARY spine binds to
			// its declaring type's static free function with $r as the
			// receiver (IS-A the owner via extends). A secondary-component
			// delegation falls through to the component's own dispatch (its
			// method may be a member or a free function of the component type).
			if p.module.isFamilyInheritedMethod(familyType, delegate.Name) {
				valueEntry := fmt.Sprintf("%s: ($r: %s, ...$a: goif$.DropFirst<Parameters<typeof %s>>) => %s($r, ...$a)",
					slot, self, target, target)
				pointerEntry := fmt.Sprintf("%s: ($r: (%s | undefined), ...$a: goif$.DropFirst<Parameters<typeof %s>>) => %s(gort$.goNilCheck<%s>($r), ...$a)",
					slot, self, target, target, self)
				if delegate.ValueReceiver {
					valueSet = append(valueSet, valueEntry)
				}
				pointerSet = append(pointerSet, pointerEntry)
				continue
			}
		}
		chain := chainOver("$r", delegate)
		// Promoted adapters spell their exact parameters through the
		// declaring method's generated function type (Parameters<> minus
		// the receiver) — exact and erased-free. The property is the
		// method's canonical SLOT (bare name unless two same-spelled
		// unexported methods from different packages both promote); an empty
		// slot is a construction defect, not a spelling fallback.
		slot := requireIdentity(delegate.Slot, "vtable slot for promoted method "+delegate.Name)
		if p.module.MethodEmissionFor(goid.Method(targetPkg, targetType, delegate.Name)) == plan.MethodOrdinaryNilChecked {
			// The declaring method is an ordinary class member: the
			// adapter calls member-shaped and types its parameters
			// through the class member itself.
			declClass, err := p.module.typeSymbol(targetPkg, targetType)
			if err != nil {
				return nil, nil, err
			}
			member := tsName(delegate.Name)
			valueEntry := fmt.Sprintf("%s: ($r: %s, ...$a: Parameters<%s[%q]>) => %s.%s(...$a)",
				slot, self, declClass, member, chain, member)
			pointerEntry := fmt.Sprintf("%s: ($r: (%s | undefined), ...$a: Parameters<%s[%q]>) => %s.%s(...$a)",
				slot, self, declClass, member, chainOver("gort$.goNilCheck<"+self+">($r)", delegate), member)
			if delegate.ValueReceiver {
				valueSet = append(valueSet, valueEntry)
			}
			pointerSet = append(pointerSet, pointerEntry)
			continue
		}
		valueEntry := fmt.Sprintf("%s: ($r: %s, ...$a: goif$.DropFirst<Parameters<typeof %s>>) => %s(%s, ...$a)",
			slot, self, target, target, chain)
		pointerEntry := fmt.Sprintf("%s: ($r: (%s | undefined), ...$a: goif$.DropFirst<Parameters<typeof %s>>) => %s(%s, ...$a)",
			slot, self, target, target, chainOver("gort$.goNilCheck<"+self+">($r)", delegate))
		if delegate.ValueReceiver {
			valueSet = append(valueSet, valueEntry)
		}
		pointerSet = append(pointerSet, pointerEntry)
	}
	return valueSet, pointerSet, nil
}

// chainOver spells the promoted field chain over a base expression,
// nil-checking each embedded-pointer step (Go's implicit dereference).
func chainOver(base string, delegate ir.PromotedDelegate) string {
	out := base
	for i, field := range delegate.Path {
		out += "." + field
		if i < len(delegate.PathPointer) && delegate.PathPointer[i] {
			out = "gort$.goNilCheck(" + out + ")"
		}
	}
	return out
}

// printIfaceDelegate emits the generated function of one method promoted
// from an embedded INTERFACE field: it takes the embedding receiver and
// dispatches through the field's closed union (the exhaustive member
// switch), exactly Go's promoted interface-method call.
func printIfaceDelegate(out *strings.Builder, module *Module, structName string, structT ir.Type, delegate ir.PromotedDelegate) error {
	if len(delegate.Path) != 1 {
		return fmt.Errorf("interface-field promotion with a multi-hop path is not yet reviewed (%s.%s)", structName, delegate.Name)
	}
	p := &printer{out: out, module: module}
	params := []string{"$r: " + tsName(structName)}
	var args []ir.Expr
	for _, parameter := range delegate.Params {
		spelled, err := p.tsType(parameter.Type)
		if err != nil {
			return err
		}
		params = append(params, parameter.Name+": "+spelled)
		args = append(args, &ir.ParamRef{Name: parameter.Name, T: parameter.Type})
	}
	result, err := p.tsFuncResultType(delegate.Results)
	if err != nil {
		return err
	}
	p.module.recordExternSymbol(tsName(structName)+"$"+delegate.Name, "")
	p.line("export function %s$%s(%s): %s {", tsName(structName), delegate.Name, joinComma(params), result)
	p.indent++
	call := &ir.IfaceCall{
		Recv:      &ir.FieldLoad{X: &ir.ParamRef{Name: "$r", T: structT}, Field: delegate.Path[0], T: delegate.IfaceType},
		Display:   delegate.Name,
		MethodKey: requireIdentity(delegate.MethodIdent, "promoted interface method "+delegate.Name),
		Args:      args,
		Results:   delegate.Results,
	}
	printed, err := p.printIfaceCall(call)
	if err != nil {
		return err
	}
	if len(delegate.Results) == 0 {
		p.line("%s;", printed)
	} else {
		p.line("return %s;", printed)
	}
	p.indent--
	p.line("}")
	return nil
}
