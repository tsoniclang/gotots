// Pointer and cell emission helpers: nil-checked reads of nilable
// operands and the proxying field-cell that gives &s.f its aliasing
// pointer when the field is not an identity carrier.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printFieldCell spells &s.f for a non-identity field as a proxying cell:
// the struct base is evaluated once (nil-checked when it is a pointer,
// exactly as Go panics on &nilptr.field) and captured, and a get/set
// accessor pair — structurally a GoCell<Elem> — reads and writes the
// live field, so a store through the pointer mutates the field.
func (p *printer) printFieldCell(n *ir.FieldCell) (string, error) {
	base, err := p.printExpr(n.Base)
	if err != nil {
		return "", err
	}
	var baseAccess, baseType string
	if n.Base.Type().Kind == ir.KindStruct {
		baseAccess = base
		baseType, err = p.tsType(n.Base.Type())
	} else {
		if n.Base.Type().Elem == nil {
			return "", fmt.Errorf("field cell base %q is not a pointer to a struct", n.Base.Type().Go)
		}
		baseAccess, err = p.nilCheckOf(base, n.Base.Type())
		if err != nil {
			return "", err
		}
		baseType, err = p.tsType(*n.Base.Type().Elem)
	}
	if err != nil {
		return "", err
	}
	elemType, err := p.tsType(n.Elem)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"((($b: %s): gort$.GoCell<%s> => ({ get v(): %s { return $b.%s; }, set v($x: %s) { $b.%s = $x; } }))(%s))",
		baseType, elemType, elemType, n.Field, elemType, n.Field, baseAccess), nil
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
