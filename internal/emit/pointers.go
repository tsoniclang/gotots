// Pointer and cell emission helpers: nil-checked reads of nilable
// operands and the reference to a field's stable per-instance cell that
// gives &s.f its exact aliasing pointer when the field is not an
// identity carrier.
package emit

import (
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printFieldCellRef spells &s.f for a non-identity cell field: the value
// is the field's stable per-instance cell itself (never a fresh proxy),
// so repeated &s.f is the same pointer and a store through it mutates the
// field. A pointer base is nil-checked, exactly as Go panics on
// &nilptr.field.
func (p *printer) printFieldCellRef(n *ir.FieldCellRef) (string, error) {
	base, err := p.printExpr(n.Base)
	if err != nil {
		return "", err
	}
	if n.Base.Type().Kind == ir.KindStruct {
		return base + "." + n.Field, nil
	}
	checked, err := p.nilCheckOf(base, n.Base.Type())
	if err != nil {
		return "", err
	}
	return checked + "." + n.Field, nil
}

// nilCheckOf spells a nil-checked read of a nilable-typed operand with an
// explicit type argument, so TypeScript's literal-undefined narrowing of
// a provably nil operand (legal Go that panics at runtime) cannot
// collapse the result type.
func (p *printer) nilCheckOf(expr string, nilable ir.Type) (string, error) {
	if nilable.Kind == ir.KindIface && nilable.TypeParamName == "" {
		union, err := p.tsType(nilable)
		if err != nil {
			return "", err
		}
		return "gort$.goNilCheck<Exclude<" + union + ", undefined>>(" + expr + ")", nil
	}
	spelled, err := p.tsType(nilable)
	if err != nil {
		return "", err
	}
	if inner, ok := strings.CutSuffix(spelled, " | undefined"); ok {
		return "gort$.goNilCheck<" + inner + ">(" + expr + ")", nil
	}
	return "gort$.goNilCheck(" + expr + ")", nil
}
