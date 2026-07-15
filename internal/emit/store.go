package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// stagedTarget is a target whose operands were pre-evaluated. Go's
// two-phase rule evaluates operands (pointer values, map operands, keys)
// and right-hand values first; panics (nil deref, nil map, bounds)
// happen at each store, so nil checks stay in the store step.
type stagedTarget struct {
	kind    string // blank | var | field | map | slice | array | pointee
	name    string // var name, or staged base/map/slice/array/pointer temp
	field   string
	keyTemp string
	// structValue marks a struct-valued store: fields overwrite in place
	// (variables, fields, pointees) or through the in-place ABI element
	// store.
	structValue bool
	// arrayValue marks a fixed-array-valued store: the slots overwrite in
	// place with the composed element callback, so element aliases and
	// slice views observe the store.
	arrayValue string
	// nilCheckBase marks a pointer base dereferenced at store time.
	nilCheckBase bool
	// keyedMap routes the store through the composite-key carrier.
	keyedMap bool
}

// arrayValueCallback spells the goArraySetAll element callback when the
// stored type is a fixed array, else "".
func (p *printer) arrayValueCallback(t ir.Type) (string, error) {
	if t.Kind != ir.KindArray {
		return "", nil
	}
	return p.arrayElemSet(*t.Elem)
}

func (p *printer) stageTarget(target ir.Target) (stagedTarget, error) {
	switch t := target.(type) {
	case ir.BlankTarget:
		return stagedTarget{kind: "blank"}, nil
	case ir.VarTarget:
		setElem, err := p.arrayValueCallback(t.T)
		if err != nil {
			return stagedTarget{}, err
		}
		return stagedTarget{kind: "var", name: tsName(t.Name),
			structValue: t.T.Kind == ir.KindStruct, arrayValue: setElem}, nil
	case *ir.FieldTarget:
		base, err := p.printExpr(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		baseTemp := p.temp()
		p.line("const %s = %s;", baseTemp, base)
		setElem, err := p.arrayValueCallback(t.T)
		if err != nil {
			return stagedTarget{}, err
		}
		return stagedTarget{kind: "field", name: baseTemp, field: t.Field,
			structValue: t.T.Kind == ir.KindStruct, arrayValue: setElem,
			nilCheckBase: t.X.Type().Kind != ir.KindStruct}, nil
	case *ir.PointeeTarget:
		pointer, err := p.printExpr(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		pointerTemp := p.temp()
		p.line("const %s = %s;", pointerTemp, pointer)
		return stagedTarget{kind: "pointee", name: pointerTemp, structValue: true, nilCheckBase: true}, nil
	case *ir.MapTarget:
		mapExpr, err := p.printExpr(t.Map)
		if err != nil {
			return stagedTarget{}, err
		}
		mapTemp := p.temp()
		p.line("const %s = %s;", mapTemp, mapExpr)
		key, err := p.printExpr(t.Key)
		if err != nil {
			return stagedTarget{}, err
		}
		keyTemp := p.temp()
		p.line("const %s = %s;", keyTemp, key)
		return stagedTarget{kind: "map", name: mapTemp, keyTemp: keyTemp,
			keyedMap: t.Map.Type().Key.Kind == ir.KindStruct}, nil
	case *ir.SliceTarget:
		sliceExpr, err := p.printExpr(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		sliceTemp := p.temp()
		p.line("const %s = %s;", sliceTemp, sliceExpr)
		index, err := p.printExpr(t.Index)
		if err != nil {
			return stagedTarget{}, err
		}
		indexTemp := p.temp()
		p.line("const %s = %s;", indexTemp, index)
		structElem := t.X.Type().Elem != nil && t.X.Type().Elem.Kind == ir.KindStruct
		setElem := ""
		if t.X.Type().Elem != nil {
			if setElem, err = p.arrayValueCallback(*t.X.Type().Elem); err != nil {
				return stagedTarget{}, err
			}
		}
		return stagedTarget{kind: "slice", name: sliceTemp, keyTemp: indexTemp,
			structValue: structElem, arrayValue: setElem}, nil
	case *ir.ArrayTarget:
		arrayExpr, err := p.printExpr(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		arrayTemp := p.temp()
		p.line("const %s = %s;", arrayTemp, arrayExpr)
		index, err := p.printExpr(t.Index)
		if err != nil {
			return stagedTarget{}, err
		}
		indexTemp := p.temp()
		p.line("const %s = %s;", indexTemp, index)
		staged := stagedTarget{kind: "array", name: arrayTemp, keyTemp: indexTemp,
			structValue: t.X.Type().Elem.Kind == ir.KindStruct}
		if staged.arrayValue, err = p.arrayValueCallback(*t.X.Type().Elem); err != nil {
			return stagedTarget{}, err
		}
		return staged, nil
	}
	return stagedTarget{}, fmt.Errorf("no staging for target %T", target)
}

func (s stagedTarget) store(p *printer, value string) error {
	base := s.name
	if s.nilCheckBase {
		base = "gort$.goNilCheck(" + s.name + ")"
	}
	switch s.kind {
	case "blank":
		p.line("void (%s);", value)
	case "var":
		switch {
		case s.structValue:
			p.line("%s.goSet$(%s);", s.name, value)
		case s.arrayValue != "":
			p.line("gosl$.goArraySetAll(%s, %s, %s);", s.name, value, s.arrayValue)
		default:
			p.line("%s = %s;", s.name, value)
		}
	case "field":
		switch {
		case s.structValue:
			p.line("%s.%s.goSet$(%s);", base, s.field, value)
		case s.arrayValue != "":
			p.line("gosl$.goArraySetAll(%s.%s, %s, %s);", base, s.field, value, s.arrayValue)
		default:
			p.line("%s.%s = %s;", base, s.field, value)
		}
	case "pointee":
		p.line("%s.goSet$(%s);", base, value)
	case "map":
		if s.keyedMap {
			p.line("gort$.goKMapSet(%s, %s, %s);", s.name, s.keyTemp, value)
		} else {
			p.line("gort$.goMapSet(%s, %s, %s);", s.name, s.keyTemp, value)
		}
	case "slice":
		switch {
		case s.structValue:
			p.line("gosl$.goSliceSetStruct(%s, %s, %s);", s.name, s.keyTemp, value)
		case s.arrayValue != "":
			p.line("gosl$.goSliceSetArray(%s, %s, %s, %s);", s.name, s.keyTemp, value, s.arrayValue)
		default:
			p.line("gosl$.goSliceSet(%s, %s, %s);", s.name, s.keyTemp, value)
		}
	case "array":
		p.printArrayElemStore(s.name, s.keyTemp, value, s)
	}
	return nil
}

// printArrayElemStore emits one bounds-checked element store into a
// fixed array: struct and nested-array elements overwrite in place.
func (p *printer) printArrayElemStore(array, index, value string, s stagedTarget) {
	switch {
	case s.structValue:
		p.line("gosl$.goArrayElemSetStruct(%s, %s, %s);", array, index, value)
	case s.arrayValue != "":
		// goArrayGet bounds-checks the slot, then the nested array
		// overwrites in place.
		p.line("gosl$.goArraySetAll(gosl$.goArrayGet(%s, %s), %s, %s);", array, index, value, s.arrayValue)
	default:
		p.line("gosl$.goArrayElemSet(%s, %s, %s);", array, index, value)
	}
}

// printStore emits a single-target store without staging (single
// assignments evaluate left-to-right naturally). Struct-valued variable
// and field stores overwrite fields in place — Go writes the value's
// memory, so every alias observes the store; struct slice elements go
// through the bounds-checked in-place ABI store.
func (p *printer) printStore(target ir.Target, value string) error {
	switch t := target.(type) {
	case ir.BlankTarget:
		p.line("void (%s);", value)
		return nil
	case ir.VarTarget:
		if t.T.Kind == ir.KindStruct {
			p.line("%s.goSet$(%s);", tsName(t.Name), value)
			return nil
		}
		if t.T.Kind == ir.KindArray {
			setElem, err := p.arrayElemSet(*t.T.Elem)
			if err != nil {
				return err
			}
			p.line("gosl$.goArraySetAll(%s, %s, %s);", tsName(t.Name), value, setElem)
			return nil
		}
		p.line("%s = %s;", tsName(t.Name), value)
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
			case t.T.Kind == ir.KindStruct:
				p.line("%s.%s.goSet$(%s);", base, t.Field, value)
			case setElem != "":
				p.line("gosl$.goArraySetAll(%s.%s, %s, %s);", base, t.Field, value, setElem)
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
		switch {
		case t.T.Kind == ir.KindStruct:
			p.line("gort$.goNilCheck(%s).%s.goSet$(%s);", baseTemp, t.Field, valueTemp)
		case setElem != "":
			p.line("gosl$.goArraySetAll(gort$.goNilCheck(%s).%s, %s, %s);", baseTemp, t.Field, valueTemp, setElem)
		default:
			p.line("gort$.goNilCheck(%s).%s = %s;", baseTemp, t.Field, valueTemp)
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
		p.line("gort$.%s(%s, %s, %s);", mapHelper("goMapSet", t.Map), mapExpr, key, value)
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
		p.line("gosl$.goSliceSet(%s, %s, %s);", sliceExpr, index, value)
		return nil
	case *ir.PointeeTarget:
		pointer, err := p.printExpr(t.X)
		if err != nil {
			return err
		}
		pointerTemp := p.temp()
		p.line("const %s = %s;", pointerTemp, pointer)
		valueTemp := p.temp()
		p.line("const %s = %s;", valueTemp, value)
		p.line("gort$.goNilCheck(%s).goSet$(%s);", pointerTemp, valueTemp)
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
		staged := stagedTarget{structValue: t.X.Type().Elem.Kind == ir.KindStruct}
		if staged.arrayValue, err = p.arrayValueCallback(*t.X.Type().Elem); err != nil {
			return err
		}
		p.printArrayElemStore(arrayTemp, indexTemp, valueTemp, staged)
		return nil
	}
	return fmt.Errorf("no emission for target %T", target)
}
