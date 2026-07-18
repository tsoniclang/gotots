// Interface equality emission: one equality function per closed union,
// doing Go's interface == by exact per-member narrowing (no erased
// payload). All equality sites (a == b, generic == operations, interface
// struct fields) call it.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

var identityPlan = &ir.EqPlan{Kind: ir.EqIdentity}

// ifaceEqFn generates one union's equality function: Go's interface == by
// exact per-member narrowing — no payload is ever recovered through an
// erased type. Two nil interfaces are equal; different dynamic types are
// unequal; same dynamic type compares by that member's recursive typed
// equality plan (=== for pointers and primitive carriers, the type's own
// goEq$ for comparable value structs, recursive element-wise for
// comparable value arrays, its own union equality for interface elements,
// a runtime panic for an uncomparable dynamic type, and fail-closed for
// an external value — the panic display read from the box's own rtti
// token, never an erased payload).
func (p *printer) ifaceEqFn(t ir.Type, name string) string {
	var cases []string
	for _, member := range p.retainedMembers(t) {
		cases = append(cases, memberEqCase(member.K, member.Eq))
	}
	if t.IfaceEmpty {
		for _, member := range predeclaredMembers {
			cases = append(cases, memberEqCase("p:"+member.name, identityPlan))
		}
		for _, composite := range p.module.BoxedComposites {
			if p.referencesWithheldType(composite.T) {
				continue
			}
			cases = append(cases, memberEqCase("c:"+composite.Canon, composite.Eq))
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "export function %s$eq(a: %s, b: %s): boolean {\n", name, name, name)
	if len(cases) == 0 {
		// A union with no retained members admits only the nil interface:
		// equality is nil-ness. (The switch would read .k on `never`.)
		b.WriteString("  return a === b;\n}\n")
		return b.String()
	}
	b.WriteString("  if (a === undefined || b === undefined) return a === b;\n")
	b.WriteString("  if (a.k !== b.k) return false;\n")
	b.WriteString("  switch (a.k) {\n")
	for _, c := range cases {
		b.WriteString("    " + c + "\n")
	}
	// a.k === b.k and every member is enumerated, so the default is
	// unreachable under correct generation (TypeScript narrows a to never
	// here) — it fails closed with the union's identity rather than
	// silently returning false, which would hide a universe-completeness
	// defect as an equality answer.
	fmt.Fprintf(&b, "    default: return gort$.goPanicUnreachableType(%q);\n", name)
	b.WriteString("  }\n}\n")
	return b.String()
}

// memberEqCase spells one switch case of a union equality: the second
// `b.k === K` narrows b's payload to the same exact member (a.k === b.k
// is already established), so the comparison touches only exact types. An
// uncomparable or external TOP-LEVEL dynamic type panics/fails-closed
// BEFORE any value read, using the box's own exact runtime display (a.r.d);
// every other case applies the member's recursive plan. A nil plan is a
// construction-invariant violation (eqPlan returns plan-or-error), never a
// silent === default.
func memberEqCase(k string, plan *ir.EqPlan) string {
	lit := fmt.Sprintf("%q", k)
	if plan == nil {
		panic(fmt.Sprintf("emit: interface union member %q carries no equality plan", k))
	}
	switch plan.Kind {
	case ir.EqUncomparable:
		return fmt.Sprintf("case %s: return goif$.goPanicUncomparable(a.r.d);", lit)
	case ir.EqExternal:
		return fmt.Sprintf("case %s: return goif$.goPanicExternalEq(a.r.d);", lit)
	}
	return fmt.Sprintf("case %s: return b.k === %s ? %s : false;", lit, lit, emitEqPlan(plan, "a.v", "b.v"))
}

// emitEqPlan spells one recursive equality plan comparing a and b. It is a
// TOTAL switch over the closed EqKind set — there is no catch-all that
// falls back to JavaScript ===. A nested uncomparable/external element
// (e.g. an array of a comparable-external struct) fails closed at element
// depth using the element type's static display. An EqInvalid or unhandled
// variant is a compiler-internal defect and panics (fail closed at
// generation), never emits wrong code.
func emitEqPlan(plan *ir.EqPlan, a, b string) string {
	if plan == nil {
		panic("emit: nil equality plan")
	}
	switch plan.Kind {
	case ir.EqIdentity:
		return a + " === " + b
	case ir.EqGoEq:
		return a + ".goEq$(" + b + ")"
	case ir.EqArray:
		return "gosl$.goArrayEqualWith(" + a + ", " + b + ", ($x, $y) => " + emitEqPlan(plan.Elem, "$x", "$y") + ")"
	case ir.EqIface:
		// An interface array element compares through its own union equality;
		// the alias name derives from the SAME shared full-digest helper as
		// the alias declaration, so the reference always resolves.
		return ifaceAliasName(plan.IfaceID) + "$eq(" + a + ", " + b + ")"
	case ir.EqUncomparable:
		return fmt.Sprintf("goif$.goPanicUncomparable(%q)", plan.Display)
	case ir.EqExternal:
		return fmt.Sprintf("goif$.goPanicExternalEq(%q)", plan.Display)
	}
	panic(fmt.Sprintf("emit: unhandled EqKind %d", plan.Kind))
}
