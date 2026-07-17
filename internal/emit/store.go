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
	// nilCheckBase marks a pointer base dereferenced at store time;
	// baseNilable is its nilable type for the explicit-generic check
	// (defeats literal-undefined narrowing on provably nil operands).
	nilCheckBase bool
	baseNilable  ir.Type
	// keyedMap routes the store through the composite-key carrier.
	keyedMap bool
	// externSet, when set, is the typed goSet$ stub reference for an
	// external-valued in-place store.
	externSet string
	// cell marks a field target stored as a stable per-instance cell: the
	// value is read and written through the cell (.v), so the field keeps
	// a stable address across the store.
	cell bool
}

// externSetCallee spells the typed goSet$ stub reference when the
// stored type is external, else "".
func mustExternSet(p *printer, t ir.Type) string {
	callee, err := p.externSetCallee(t)
	if err != nil {
		return ""
	}
	return callee
}

func (p *printer) externSetCallee(t ir.Type) (string, error) {
	if t.Kind != ir.KindExternal {
		return "", nil
	}
	return p.module.symbol(t.Pkg, externSetSymbol(t.Named))
}

// arrayValueCallback spells the goArraySetAll element callback when the
// stored type is a fixed array, else "".
func (p *printer) arrayValueCallback(t ir.Type) (string, error) {
	if t.Kind != ir.KindArray {
		return "", nil
	}
	return p.arrayElemSet(*t.Elem)
}

// operandSlot spells one compound-target operand: a pure operand
// (variable or folded constant) re-emits directly, so a compound
// assignment observes its post-RHS value exactly as gc does; an impure
// operand (a call, index, etc.) evaluates into a temp before the RHS,
// preserving lexical evaluation order.
func (p *printer) operandSlot(e ir.Expr) (string, error) {
	printed, err := p.printExpr(e)
	if err != nil {
		return "", err
	}
	if pureOperandExpr(e) {
		return printed, nil
	}
	temp := p.temp()
	p.line("const %s = %s;", temp, printed)
	return temp, nil
}

// pureOperandExpr reports whether re-evaluating one operand is exact and
// observes the current value: a local variable read or a folded
// constant. Package variables (live ESM bindings) also qualify.
func pureOperandExpr(e ir.Expr) bool {
	switch e.(type) {
	case *ir.VarRef, *ir.Const, *ir.BoxedLoad:
		return true
	}
	return false
}

// stageCompoundTarget stages one compound-assignment target for gc's
// evaluation order: impure operands evaluate (into temps) before the
// RHS at the call site; pure operands re-emit at load and store, so the
// container/base/pointer is read AFTER the RHS.
func (p *printer) stageCompoundTarget(target ir.Target) (stagedTarget, error) {
	switch t := target.(type) {
	case ir.VarTarget:
		return stagedTarget{kind: "var", name: tsName(t.Name), structValue: t.T.Kind == ir.KindStruct,
			externSet: mustExternSet(p, t.T)}, nil
	case ir.BoxedTarget:
		return stagedTarget{kind: "boxed", name: t.Cell}, nil
	case *ir.FieldTarget:
		base, err := p.operandSlot(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		return stagedTarget{kind: "field", name: base, field: t.Field,
			structValue: t.T.Kind == ir.KindStruct, externSet: mustExternSet(p, t.T),
			nilCheckBase: t.X.Type().Kind != ir.KindStruct,
			baseNilable:  t.X.Type(), cell: t.Cell}, nil
	case *ir.PointeeTarget:
		pointer, err := p.operandSlot(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		staged := stagedTarget{kind: "pointee", name: pointer, nilCheckBase: true, baseNilable: t.X.Type()}
		if elem := t.X.Type().Elem; elem != nil {
			staged.structValue = elem.Kind == ir.KindStruct
			staged.externSet = mustExternSet(p, *elem)
		}
		return staged, nil
	case *ir.MapTarget:
		mapOp, err := p.operandSlot(t.Map)
		if err != nil {
			return stagedTarget{}, err
		}
		key, err := p.operandSlot(t.Key)
		if err != nil {
			return stagedTarget{}, err
		}
		return stagedTarget{kind: "map", name: mapOp, keyTemp: key,
			keyedMap: t.Map.Type().Key.Kind == ir.KindStruct}, nil
	case *ir.SliceTarget:
		sliceOp, err := p.operandSlot(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		index, err := p.operandSlot(t.Index)
		if err != nil {
			return stagedTarget{}, err
		}
		structElem := t.X.Type().Elem != nil && t.X.Type().Elem.Kind == ir.KindStruct
		staged := stagedTarget{kind: "slice", name: sliceOp, keyTemp: index, structValue: structElem}
		if p.nativeSlice(t.X) {
			staged.kind = "array"
		}
		return staged, nil
	case *ir.ArrayTarget:
		arrayOp, err := p.operandSlot(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		index, err := p.operandSlot(t.Index)
		if err != nil {
			return stagedTarget{}, err
		}
		return stagedTarget{kind: "array", name: arrayOp, keyTemp: index,
			structValue: t.X.Type().Elem.Kind == ir.KindStruct}, nil
	}
	return stagedTarget{}, fmt.Errorf("no compound staging for target %T", target)
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
			structValue: t.T.Kind == ir.KindStruct, arrayValue: setElem,
			externSet: mustExternSet(p, t.T)}, nil
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
			externSet:    mustExternSet(p, t.T),
			nilCheckBase: t.X.Type().Kind != ir.KindStruct,
			baseNilable:  t.X.Type(), cell: t.Cell}, nil
	case *ir.PointeeTarget:
		pointer, err := p.printExpr(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		pointerTemp := p.temp()
		p.line("const %s = %s;", pointerTemp, pointer)
		staged := stagedTarget{kind: "pointee", name: pointerTemp, nilCheckBase: true, baseNilable: t.X.Type()}
		if elem := t.X.Type().Elem; elem != nil {
			staged.structValue = elem.Kind == ir.KindStruct
			if staged.arrayValue, err = p.arrayValueCallback(*elem); err != nil {
				return stagedTarget{}, err
			}
			staged.externSet = mustExternSet(p, *elem)
		}
		return staged, nil
	case ir.BoxedTarget:
		return stagedTarget{kind: "boxed", name: t.Cell}, nil
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
		staged := stagedTarget{kind: "slice", name: sliceTemp, keyTemp: indexTemp,
			structValue: structElem, arrayValue: setElem}
		if p.nativeSlice(t.X) {
			staged.kind = "array"
		}
		return staged, nil
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

// load spells the staged location's current value; reading happens
// after the right-hand side evaluates and before the store, exactly
// Go's compound sequence.
func (s stagedTarget) load(p *printer, operandT ir.Type) (string, error) {
	base := s.name
	if s.nilCheckBase {
		checked, err := p.nilCheckOf(s.name, s.baseNilable)
		if err != nil {
			return "", err
		}
		base = checked
	}
	switch s.kind {
	case "var":
		return s.name, nil
	case "boxed":
		return s.name + ".v", nil
	case "field":
		if s.cell {
			return base + "." + s.field + ".v", nil
		}
		return base + "." + s.field, nil
	case "pointee":
		return base + ".v", nil
	case "map":
		zero, err := p.zeroLiteral(operandT)
		if err != nil {
			return "", err
		}
		if s.keyedMap {
			return "gort$.goKMapGet(" + s.name + ", " + s.keyTemp + ", " + zero + ")", nil
		}
		return "gort$.goMapGet(" + s.name + ", " + s.keyTemp + ", " + zero + ")", nil
	case "slice":
		return "gosl$.goSliceGet(" + s.name + ", " + s.keyTemp + ")", nil
	case "array":
		return "gosl$.goArrayGet(" + s.name + ", " + s.keyTemp + ")", nil
	}
	return "", fmt.Errorf("no load for staged target kind %q", s.kind)
}

func (s stagedTarget) store(p *printer, value string) error {
	base := s.name
	if s.nilCheckBase {
		checked, err := p.nilCheckOf(s.name, s.baseNilable)
		if err != nil {
			return err
		}
		base = checked
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
		case s.externSet != "":
			p.line("%s(%s, %s);", s.externSet, s.name, value)
		default:
			p.line("%s = %s;", s.name, value)
		}
	case "field":
		switch {
		case s.cell:
			// A cell field keeps its stable storage: only the held value is
			// overwritten, so a previously taken &s.f still observes the write.
			p.line("%s.%s.v = %s;", base, s.field, value)
		case s.structValue:
			p.line("%s.%s.goSet$(%s);", base, s.field, value)
		case s.arrayValue != "":
			p.line("gosl$.goArraySetAll(%s.%s, %s, %s);", base, s.field, value, s.arrayValue)
		case s.externSet != "":
			p.line("%s(%s.%s, %s);", s.externSet, base, s.field, value)
		default:
			p.line("%s.%s = %s;", base, s.field, value)
		}
	case "pointee":
		switch {
		case s.structValue:
			p.line("%s.goSet$(%s);", base, value)
		case s.arrayValue != "":
			p.line("gosl$.goArraySetAll(%s, %s, %s);", base, value, s.arrayValue)
		case s.externSet != "":
			p.line("%s(%s, %s);", s.externSet, base, value)
		default:
			p.line("%s.v = %s;", base, value)
		}
	case "boxed":
		p.line("%s.v = %s;", s.name, value)
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
		if callee := mustExternSet(p, t.T); callee != "" {
			p.line("%s(%s, %s);", callee, tsName(t.Name), value)
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
		staged := stagedTarget{structValue: t.X.Type().Elem.Kind == ir.KindStruct}
		if staged.arrayValue, err = p.arrayValueCallback(*t.X.Type().Elem); err != nil {
			return err
		}
		p.printArrayElemStore(arrayTemp, indexTemp, valueTemp, staged)
		return nil
	}
	return fmt.Errorf("no emission for target %T", target)
}
