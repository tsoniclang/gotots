// Fixed-array emission: native arrays with value semantics — copies at
// Go copy boundaries through composed element-clone callbacks, in-place
// whole-value stores, exact bounds panics, and slice views sharing the
// array's storage.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// arrayElemClone spells the element-clone callback for goArrayClone:
// undefined for direct-copy elements, a goClone$ call for struct
// elements, and a recursive composition for nested arrays.
func (p *printer) arrayElemClone(elem ir.Type) (string, error) {
	switch elem.Kind {
	case ir.KindStruct:
		spelled, err := p.tsType(elem)
		if err != nil {
			return "", err
		}
		return "(v: " + spelled + ") => v.goClone$()", nil
	case ir.KindArray:
		inner, err := p.arrayElemClone(*elem.Elem)
		if err != nil {
			return "", err
		}
		spelled, err := p.tsType(elem)
		if err != nil {
			return "", err
		}
		return "(v: " + spelled + ") => gosl$.goArrayClone(v, " + inner + ")", nil
	}
	return "undefined", nil
}

// arrayElemSet spells the in-place element-store callback for
// goArraySetAll, composed the same way.
func (p *printer) arrayElemSet(elem ir.Type) (string, error) {
	switch elem.Kind {
	case ir.KindStruct:
		spelled, err := p.tsType(elem)
		if err != nil {
			return "", err
		}
		return "(d: " + spelled + ", s: " + spelled + ") => d.goSet$(s)", nil
	case ir.KindArray:
		inner, err := p.arrayElemSet(*elem.Elem)
		if err != nil {
			return "", err
		}
		spelled, err := p.tsType(elem)
		if err != nil {
			return "", err
		}
		return "(d: " + spelled + ", s: " + spelled + ") => gosl$.goArraySetAll(d, s, " + inner + ")", nil
	}
	return "undefined", nil
}

// arrayZeroFactory spells the fresh-zero factory for one array type.
func (p *printer) arrayZeroFactory(t ir.Type) (string, error) {
	elemZero, err := p.zeroLiteral(*t.Elem)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gosl$.goArrayZero(%d, () => %s)", t.ArrayLen, elemZero), nil
}

// printArrayExpr emits the array-specific expression nodes; ok reports
// whether the node was one of them.
func (p *printer) printArrayExpr(e ir.Expr) (string, bool, error) {
	switch n := e.(type) {
	case *ir.ArrayLit:
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", true, err
		}
		return "[" + values + "]", true, nil
	case *ir.ArrayGet:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		index, err := p.printExpr(n.Index)
		if err != nil {
			return "", true, err
		}
		return "gosl$.goArrayGet(" + x + ", " + index + ")", true, nil
	case *ir.ArrayLenExpr:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		return "gosl$.goArrayLen(" + x + ")", true, nil
	case *ir.ArrayEqual:
		left, err := p.printExpr(n.L)
		if err != nil {
			return "", true, err
		}
		right, err := p.printExpr(n.R)
		if err != nil {
			return "", true, err
		}
		if n.Negate {
			return "(!gosl$.goArrayEqual(" + left + ", " + right + "))", true, nil
		}
		return "gosl$.goArrayEqual(" + left + ", " + right + ")", true, nil
	case *ir.ArraySliceView:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		low := "0"
		if n.Low != nil {
			if low, err = p.printExpr(n.Low); err != nil {
				return "", true, err
			}
		}
		high := "undefined"
		if n.High != nil {
			if high, err = p.printExpr(n.High); err != nil {
				return "", true, err
			}
		}
		return "gosl$.goSliceFromArray(" + x + ", " + low + ", " + high + ")", true, nil
	}
	return "", false, nil
}
