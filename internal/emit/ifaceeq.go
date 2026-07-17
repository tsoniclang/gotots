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

// ifaceEqFn generates one union's equality function: Go's interface == by
// exact per-member narrowing — no payload is ever recovered through an
// erased type. Two nil interfaces are equal; different dynamic types are
// unequal; same dynamic type compares by that member's exact operation
// (=== for pointers and primitive carriers, the type's own goEq$ for
// comparable value structs, element-wise for comparable value arrays, a
// runtime panic for an uncomparable dynamic type, and fail-closed for an
// external value's unknown comparability — its display read from the
// box's own rtti token, never an erased payload).
func (p *printer) ifaceEqFn(t ir.Type, name string) string {
	var cases []string
	for _, member := range p.retainedMembers(t) {
		cases = append(cases, memberEqCase(member.K, member.EqMode, member.ArrayElemEq))
	}
	if t.IfaceEmpty {
		for _, member := range predeclaredMembers {
			cases = append(cases, memberEqCase("p:"+member.name, "identity", ""))
		}
		for _, composite := range p.module.BoxedComposites {
			if p.referencesWithheldType(composite.T) {
				continue
			}
			cases = append(cases, memberEqCase("c:"+composite.Canon, composite.EqMode, composite.ArrayElemEq))
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
	b.WriteString("    default: return false;\n")
	b.WriteString("  }\n}\n")
	return b.String()
}

// memberEqCase spells one switch case of a union equality: the second
// `b.k === K` narrows b's payload to the same exact member (a.k === b.k
// is already established), so the comparison touches only exact types.
func memberEqCase(k, eqMode, arrayElemEq string) string {
	lit := fmt.Sprintf("%q", k)
	switch eqMode {
	case "goEq":
		return fmt.Sprintf("case %s: return b.k === %s ? a.v.goEq$(b.v) : false;", lit, lit)
	case "arrayEq":
		elemCmp := "$x === $y"
		if arrayElemEq == "goEq" {
			elemCmp = "$x.goEq$($y)"
		}
		return fmt.Sprintf("case %s: return b.k === %s ? gosl$.goArrayEqualWith(a.v, b.v, ($x, $y) => %s) : false;", lit, lit, elemCmp)
	case "uncomparable":
		return fmt.Sprintf("case %s: return goif$.goPanicUncomparable(a.r.d);", lit)
	case "external":
		return fmt.Sprintf("case %s: return goif$.goPanicExternalEq(a.r.d);", lit)
	}
	return fmt.Sprintf("case %s: return b.k === %s && a.v === b.v;", lit, lit)
}
