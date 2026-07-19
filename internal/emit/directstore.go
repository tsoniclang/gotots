// Direct (unstaged) single-target stores: a plain variable, field,
// pointee, map, or slice element assigned left-to-right with no
// pre-evaluated operands. varStoreExpr is the single variable-store
// rendering shared with the native for-header clause.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

func (p *printer) varStoreExpr(t ir.VarTarget, value string) (string, error) {
	name := tsName(t.Name)
	if t.T.Kind == ir.KindStruct {
		return name + ".goSet$(" + value + ")", nil
	}
	if t.T.Kind == ir.KindArray {
		setElem, err := p.arrayElemSet(*t.T.Elem)
		if err != nil {
			return "", err
		}
		return "gosl$.goArraySetAll(" + name + ", " + value + ", " + setElem + ")", nil
	}
	if callee := mustExternSet(p, t.T); callee != "" {
		return callee + "(" + name + ", " + value + ")", nil
	}
	return name + " = " + value, nil
}

func (p *printer) printStore(target ir.Target, value string) error {
	switch t := target.(type) {
	case ir.BlankTarget:
		p.line("void (%s);", value)
		return nil
	case ir.VarTarget:
		expr, err := p.varStoreExpr(t, value)
		if err != nil {
			return err
		}
		p.line("%s;", expr)
		return nil
	case *ir.FieldTarget:
		base, err := p.printExpr(t.X)
		if err != nil {
			return err
		}
		setElem, err := p.arrayValueCallback(t.T)
		if err != nil {
			return err
		}
		if t.X.Type().Kind == ir.KindStruct {
			// A struct-value base cannot panic; the direct member store
			// already evaluates base, then value, then stores.
			switch {
			case p.paramSetOf(t.T) != "":
				set := p.paramSetOf(t.T)
				p.line("if (%s === undefined) { %s.%s = %s; } else { %s(%s.%s, %s); }",
					set, base, t.Field, value, set, base, t.Field, value)
			case t.Cell:
				p.line("%s.%s.v = %s;", base, t.Field, value)
			case t.T.Kind == ir.KindStruct:
				p.line("%s.%s.goSet$(%s);", base, t.Field, value)
			case setElem != "":
				p.line("gosl$.goArraySetAll(%s.%s, %s, %s);", base, t.Field, value, setElem)
			case mustExternSet(p, t.T) != "":
				p.line("%s(%s.%s, %s);", mustExternSet(p, t.T), base, t.Field, value)
			default:
				p.line("%s.%s = %s;", base, t.Field, value)
			}
			return nil
		}
		// Go evaluates the pointer operand and the value first and panics
		// at the store; the temps keep the nil check after the value.
		baseTemp := p.temp()
		p.line("const %s = %s;", baseTemp, base)
		valueTemp := p.temp()
		p.line("const %s = %s;", valueTemp, value)
		checkedBase, err := p.nilCheckOf(baseTemp, t.X.Type())
		if err != nil {
			return err
		}
		switch {
		case p.paramSetOf(t.T) != "":
			set := p.paramSetOf(t.T)
			p.line("if (%s === undefined) { %s.%s = %s; } else { %s(%s.%s, %s); }",
				set, checkedBase, t.Field, valueTemp, set, checkedBase, t.Field, valueTemp)
		case t.Cell:
			// A cell field keeps its stable storage: only the held value is
			// overwritten, so a previously taken &s.f still observes the write.
			p.line("%s.%s.v = %s;", checkedBase, t.Field, valueTemp)
		case t.T.Kind == ir.KindStruct:
			p.line("%s.%s.goSet$(%s);", checkedBase, t.Field, valueTemp)
		case setElem != "":
			p.line("gosl$.goArraySetAll(%s.%s, %s, %s);", checkedBase, t.Field, valueTemp, setElem)
		case mustExternSet(p, t.T) != "":
			p.line("%s(%s.%s, %s);", mustExternSet(p, t.T), checkedBase, t.Field, valueTemp)
		default:
			p.line("%s.%s = %s;", checkedBase, t.Field, valueTemp)
		}
		return nil
	case *ir.MapTarget:
		mapExpr, err := p.printExpr(t.Map)
		if err != nil {
			return err
		}
		key, err := p.printExpr(t.Key)
		if err != nil {
			return err
		}
		if t.Map.Type().Key != nil {
			encoder, err := p.ifaceKeyEncoder(t.Map.Type())
			if err != nil {
				return err
			}
			if encoder != "" {
				p.line("gort$.goEMapSet(%s, %s, %s, %s);", mapExpr, key, value, encoder)
				return nil
			}
		}
		p.line("gort$.%s(%s, %s, %s);", p.mapHelper("goMapSet", t.Map), mapExpr, key, value)
		return nil
	case *ir.SliceTarget:
		sliceExpr, err := p.printExpr(t.X)
		if err != nil {
			return err
		}
		index, err := p.printExpr(t.Index)
		if err != nil {
			return err
		}
		if t.X.Type().Elem != nil && p.paramSetOf(*t.X.Type().Elem) != "" {
			p.line("gosl$.goSliceSetWith(%s, %s, %s, %s);", sliceExpr, index, value, p.paramSetOf(*t.X.Type().Elem))
			return nil
		}
		if t.X.Type().Elem != nil && t.X.Type().Elem.Kind == ir.KindStruct {
			p.line("gosl$.goSliceSetStruct(%s, %s, %s);", sliceExpr, index, value)
			return nil
		}
		if t.X.Type().Elem != nil && t.X.Type().Elem.Kind == ir.KindArray {
			setElem, err := p.arrayElemSet(*t.X.Type().Elem.Elem)
			if err != nil {
				return err
			}
			p.line("gosl$.goSliceSetArray(%s, %s, %s, %s);", sliceExpr, index, value, setElem)
			return nil
		}
		if p.nativeSlice(t.X) {
			p.line("gosl$.goArrayElemSet(%s, %s, %s);", sliceExpr, index, value)
			return nil
		}
		p.line("gosl$.goSliceSet(%s, %s, %s);", sliceExpr, index, value)
		return nil
	case *ir.PointeeTarget:
		pointer, err := p.printExpr(t.X)
		if err != nil {
			return err
		}
		pointerTemp := p.temp()
		p.line("const %s = %s;", pointerTemp, pointer)
		elem := t.X.Type().Elem
		valueTemp := p.temp()
		// The value temp carries the pointee's exact type so a
		// contextually-typed right side (a bare goMapMake(), goSliceNil(),
		// or undefined nil) infers its element types here instead of
		// collapsing to unknown once the assignment target is a staged temp.
		if elem != nil {
			elemSpelled, err := p.tsType(*elem)
			if err != nil {
				return err
			}
			p.line("const %s: %s = %s;", valueTemp, elemSpelled, value)
		} else {
			p.line("const %s = %s;", valueTemp, value)
		}
		checkedPointer, err := p.nilCheckOf(pointerTemp, t.X.Type())
		if err != nil {
			return err
		}
		switch {
		case elem != nil && elem.Kind == ir.KindIface && elem.TypeParamName != "":
			// Unreachable by construction: the IR rejects pointee stores
			// through a pointer-to-type-parameter.
			return fmt.Errorf("pointee store through pointer-to-type-parameter reached emission")
		case elem == nil || elem.Kind == ir.KindStruct:
			p.line("%s.goSet$(%s);", checkedPointer, valueTemp)
		case elem.Kind == ir.KindArray:
			setElem, err := p.arrayElemSet(*elem.Elem)
			if err != nil {
				return err
			}
			p.line("gosl$.goArraySetAll(%s, %s, %s);", checkedPointer, valueTemp, setElem)
		case elem.Kind == ir.KindExternal:
			p.line("%s(%s, %s);", mustExternSet(p, *elem), checkedPointer, valueTemp)
		default:
			p.line("%s.v = %s;", checkedPointer, valueTemp)
		}
		return nil
	case ir.BoxedTarget:
		p.line("%s.v = %s;", t.Cell, value)
		return nil
	case *ir.ArrayTarget:
		arrayExpr, err := p.printExpr(t.X)
		if err != nil {
			return err
		}
		arrayTemp := p.temp()
		p.line("const %s = %s;", arrayTemp, arrayExpr)
		index, err := p.printExpr(t.Index)
		if err != nil {
			return err
		}
		indexTemp := p.temp()
		p.line("const %s = %s;", indexTemp, index)
		// The value evaluates before the bounds panic, matching Go's
		// operands-then-store order.
		valueTemp := p.temp()
		p.line("const %s = %s;", valueTemp, value)
		if set := p.paramSetOf(*t.X.Type().Elem); set != "" {
			p.line("gosl$.goArrayElemSetWith(%s, %s, %s, %s);", arrayTemp, indexTemp, valueTemp, set)
			return nil
		}
		staged := stagedTarget{structValue: t.X.Type().Elem.Kind == ir.KindStruct}
		if staged.arrayValue, err = p.arrayValueCallback(*t.X.Type().Elem); err != nil {
			return err
		}
		p.printArrayElemStore(arrayTemp, indexTemp, valueTemp, staged)
		return nil
	}
	return fmt.Errorf("no emission for target %T", target)
}
