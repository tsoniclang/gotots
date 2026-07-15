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
	kind    string // blank | var | field | map | slice | pointee
	name    string // var name, or staged base/map/slice/pointer temp
	field   string
	keyTemp string
	// structValue marks a struct-valued store: fields overwrite in place
	// (variables, fields, pointees) or through the in-place ABI element
	// store.
	structValue bool
	// nilCheckBase marks a pointer base dereferenced at store time.
	nilCheckBase bool
}

func (p *printer) stageTarget(target ir.Target) (stagedTarget, error) {
	switch t := target.(type) {
	case ir.BlankTarget:
		return stagedTarget{kind: "blank"}, nil
	case ir.VarTarget:
		return stagedTarget{kind: "var", name: t.Name, structValue: t.T.Kind == ir.KindStruct}, nil
	case *ir.FieldTarget:
		base, err := p.printExpr(t.X)
		if err != nil {
			return stagedTarget{}, err
		}
		baseTemp := p.temp()
		p.line("const %s = %s;", baseTemp, base)
		return stagedTarget{kind: "field", name: baseTemp, field: t.Field,
			structValue: t.T.Kind == ir.KindStruct, nilCheckBase: t.X.Type().Kind != ir.KindStruct}, nil
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
		return stagedTarget{kind: "map", name: mapTemp, keyTemp: keyTemp}, nil
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
		return stagedTarget{kind: "slice", name: sliceTemp, keyTemp: indexTemp, structValue: structElem}, nil
	}
	return stagedTarget{}, fmt.Errorf("no staging for target %T", target)
}

func (s stagedTarget) store(p *printer, value string) error {
	base := s.name
	if s.nilCheckBase {
		base = "gort.goNilCheck(" + s.name + ")"
	}
	switch s.kind {
	case "blank":
		p.line("void (%s);", value)
	case "var":
		if s.structValue {
			p.line("%s.goSet$(%s);", s.name, value)
		} else {
			p.line("%s = %s;", s.name, value)
		}
	case "field":
		if s.structValue {
			p.line("%s.%s.goSet$(%s);", base, s.field, value)
		} else {
			p.line("%s.%s = %s;", base, s.field, value)
		}
	case "pointee":
		p.line("%s.goSet$(%s);", base, value)
	case "map":
		p.line("gort.goMapSet(%s, %s, %s);", s.name, s.keyTemp, value)
	case "slice":
		if s.structValue {
			p.line("gosl.goSliceSetStruct(%s, %s, %s);", s.name, s.keyTemp, value)
		} else {
			p.line("gosl.goSliceSet(%s, %s, %s);", s.name, s.keyTemp, value)
		}
	}
	return nil
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
			p.line("%s.goSet$(%s);", t.Name, value)
			return nil
		}
		p.line("%s = %s;", t.Name, value)
		return nil
	case *ir.FieldTarget:
		base, err := p.printExpr(t.X)
		if err != nil {
			return err
		}
		if t.X.Type().Kind == ir.KindStruct {
			// A struct-value base cannot panic; the direct member store
			// already evaluates base, then value, then stores.
			if t.T.Kind == ir.KindStruct {
				p.line("%s.%s.goSet$(%s);", base, t.Field, value)
			} else {
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
		if t.T.Kind == ir.KindStruct {
			p.line("gort.goNilCheck(%s).%s.goSet$(%s);", baseTemp, t.Field, valueTemp)
		} else {
			p.line("gort.goNilCheck(%s).%s = %s;", baseTemp, t.Field, valueTemp)
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
		p.line("gort.goMapSet(%s, %s, %s);", mapExpr, key, value)
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
			p.line("gosl.goSliceSetStruct(%s, %s, %s);", sliceExpr, index, value)
			return nil
		}
		p.line("gosl.goSliceSet(%s, %s, %s);", sliceExpr, index, value)
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
		p.line("gort.goNilCheck(%s).goSet$(%s);", pointerTemp, valueTemp)
		return nil
	}
	return fmt.Errorf("no emission for target %T", target)
}
