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
	// Tier A: the member receiver is this-bound and non-optional — a
	// check would be dead code on a non-nullable type.
	if p.memberReceiver != "" && expr == p.memberReceiver {
		return expr, nil
	}
	// Tier B: a bare identifier already checked in this straight-line
	// region is dominator-proven non-nil; the evidence-backed assertion
	// form carries the proof without a second runtime check (Go panics
	// once, at the first dereference).
	if isBareIdent(expr) {
		if p.checkedNilables[expr] {
			return expr + "!", nil
		}
		if p.checkedNilables == nil {
			p.checkedNilables = map[string]bool{}
		}
		p.checkedNilables[expr] = true
	}
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

// isBareIdent reports whether a printed expression is a single
// identifier (the only spelling the straight-line dominator tracker
// may key on).
func isBareIdent(expr string) bool {
	if expr == "" {
		return false
	}
	for i, r := range expr {
		switch {
		case r == '_' || r == '$' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// resetNilRegion ends the current straight-line region: compound
// constructs, assignments, and closure boundaries invalidate the
// dominator proof conservatively.
func (p *printer) resetNilRegion() {
	p.checkedNilables = nil
}
