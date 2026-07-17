// Multi-result forwarding emission: the once-evaluated, per-slot
// converted carriers behind Go's f(g()) call form, for both the plain
// positional spread and the variadic tail-packing variant.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// tupleSlots spells each result slot's conversion over a bound $t tuple,
// returning the converted element expressions plus the source and result
// slot type spellings.
func (p *printer) tupleSlots(slots []ir.TupleSlot, srcTypes []ir.Type) (converted, slotTypes, resultTypes []string, err error) {
	converted = make([]string, len(slots))
	for i, slot := range slots {
		element := fmt.Sprintf("$t[%d]", i)
		switch slot.Op {
		case ir.TupleSlotBox:
			rtti, err := p.rttiRef(slot.Rtti)
			if err != nil {
				return nil, nil, nil, err
			}
			vtable, err := p.boxVtable(slot.Rtti)
			if err != nil {
				return nil, nil, nil, err
			}
			converted[i] = fmt.Sprintf("goif$.goIfaceBox(%q, %s, %s, %s)", boxDiscriminant(slot.Rtti), rtti, element, vtable)
		case ir.TupleSlotClone:
			converted[i] = element + ".goClone$()"
		default:
			converted[i] = element
		}
	}
	slotTypes = make([]string, len(srcTypes))
	resultTypes = make([]string, len(srcTypes))
	for i, t := range srcTypes {
		spelled, err := p.tsType(t)
		if err != nil {
			return nil, nil, nil, err
		}
		slotTypes[i] = spelled
		if slots[i].Op == ir.TupleSlotBox {
			target, err := p.tsType(slots[i].Target)
			if err != nil {
				return nil, nil, nil, err
			}
			resultTypes[i] = target
		} else {
			resultTypes[i] = spelled
		}
	}
	return converted, slotTypes, resultTypes, nil
}

// spreadInner reports whether args is a single multi-result spread
// argument (Go's f(g()) form, plain or variadic) and, if so, spells the
// tuple expression it spreads. The staged-arrow call forms (dynamic and
// interface calls) bind this once and spread it, so f(g()) lowers on
// every call path, not only the ordinary printArgs one.
func (p *printer) spreadInner(args []ir.Expr) (string, bool, error) {
	if len(args) != 1 {
		return "", false, nil
	}
	switch s := args[0].(type) {
	case *ir.TupleSpread:
		inner, err := p.printExpr(s.X)
		return inner, true, err
	case *ir.TupleVariadicSpread:
		inner, err := p.printTupleVariadicSpread(s)
		return inner, true, err
	}
	return "", false, nil
}

// printTupleAdapt spells a per-slot converted multi-result value: the
// inner value evaluates once, each slot converts, and the converted
// tuple is rebuilt in order.
func (p *printer) printTupleAdapt(n *ir.TupleAdapt) (string, error) {
	inner, err := p.printExpr(n.X)
	if err != nil {
		return "", err
	}
	converted, slotTypes, resultTypes, err := p.tupleSlots(n.Slots, n.SrcType)
	if err != nil {
		return "", err
	}
	return "((($t: readonly [" + joinComma(slotTypes) + "]): readonly [" + joinComma(resultTypes) + "] => [" + joinComma(converted) + "])(" + inner + "))", nil
}

// printTupleVariadicSpread spells Go's f(g()) where f is variadic: the
// converted results bind once, the leading Fixed slots forward
// positionally, and every remaining slot packs into the variadic slice —
// the whole tuple spreads into the enclosing call by the ... at the call
// site.
func (p *printer) printTupleVariadicSpread(n *ir.TupleVariadicSpread) (string, error) {
	inner, err := p.printExpr(n.X)
	if err != nil {
		return "", err
	}
	slotTypes := make([]string, len(n.SlotTypes))
	for i, t := range n.SlotTypes {
		if slotTypes[i], err = p.tsType(t); err != nil {
			return "", err
		}
	}
	sliceType, err := p.tsType(n.SliceType)
	if err != nil {
		return "", err
	}
	// The result tuple: the Fixed regular arguments, then the variadic
	// slice packed from the remaining results.
	resultTypes := append(append([]string{}, slotTypes[:n.Fixed]...), sliceType)
	elements := make([]string, 0, n.Fixed+1)
	for i := 0; i < n.Fixed; i++ {
		elements = append(elements, fmt.Sprintf("$t[%d]", i))
	}
	rest := make([]string, 0, len(n.SlotTypes)-n.Fixed)
	for i := n.Fixed; i < len(n.SlotTypes); i++ {
		rest = append(rest, fmt.Sprintf("$t[%d]", i))
	}
	if len(rest) == 0 {
		// No results remain for the variadic parameter: Go leaves it the
		// nil slice (rest == nil), exactly as a call with no variadic
		// arguments — never a fresh non-nil empty slice.
		elements = append(elements, "undefined")
	} else {
		elements = append(elements, "gosl$.goSliceFrom(["+joinComma(rest)+"])")
	}
	return "((($t: readonly [" + joinComma(slotTypes) + "]): readonly [" + joinComma(resultTypes) + "] => [" + joinComma(elements) + "])(" + inner + "))", nil
}
