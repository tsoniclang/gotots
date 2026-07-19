// Map and string expression emission: the plain and composite-key map
// carrier families, byte-string operations, and the string conversion
// family.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printCollectionExpr renders the map- and string-family expression
// nodes; ok reports whether the node was one of them.
func (p *printer) printCollectionExpr(e ir.Expr) (string, bool, error) {
	printed, err := p.collectionExpr(e)
	if err == errNotCollection {
		return "", false, nil
	}
	return printed, true, err
}

var errNotCollection = fmt.Errorf("not a collection expression")

func (p *printer) collectionExpr(e ir.Expr) (string, error) {
	switch n := e.(type) {
	case *ir.MapMake:
		hint := ""
		if n.Hint != nil {
			printed, err := p.printExpr(n.Hint)
			if err != nil {
				return "", err
			}
			hint = printed
		}
		if n.Type().Key != nil && n.Type().Key.Kind == ir.KindStruct {
			return "gort$.goKMapMake(" + hint + ")", nil
		}
		if n.Type().Key != nil && n.Type().Key.Kind == ir.KindIface && n.Type().Key.TypeParamName == "" {
			key, err := p.tsType(*n.Type().Key)
			if err != nil {
				return "", err
			}
			value, err := p.tsType(*n.Type().Elem)
			if err != nil {
				return "", err
			}
			return "gort$.goEMapMake<" + key + ", " + value + ">(" + hint + ")", nil
		}
		if n.Type().Key != nil && n.Type().Key.Kind.Float() {
			value, err := p.tsType(*n.Type().Elem)
			if err != nil {
				return "", err
			}
			return "gort$.goFMapMake<" + value + ">(" + hint + ")", nil
		}
		// Explicit type arguments: inference from a bare make has nothing
		// to infer from in a generic context (Map<unknown, unknown>), so
		// the exact key/value types always spell.
		if n.Type().Key != nil && n.Type().Elem != nil {
			key, err := p.tsType(*n.Type().Key)
			if err != nil {
				return "", err
			}
			value, err := p.tsType(*n.Type().Elem)
			if err != nil {
				return "", err
			}
			return "gort$.goMapMake<" + key + ", " + value + ">(" + hint + ")", nil
		}
		return "gort$.goMapMake(" + hint + ")", nil
	case *ir.MapFrom:
		var entries []string
		for i := range n.Keys {
			key, err := p.printExpr(n.Keys[i])
			if err != nil {
				return "", err
			}
			value, err := p.printExpr(n.Values[i])
			if err != nil {
				return "", err
			}
			entries = append(entries, "["+key+", "+value+"]")
		}
		if n.T.Key.Kind == ir.KindStruct {
			return "gort$.goKMapFrom([" + joinComma(entries) + "])", nil
		}
		if n.T.Key.Kind.Float() {
			return "gort$.goFMapFrom([" + joinComma(entries) + "])", nil
		}
		if n.T.Key.Kind == ir.KindIface && n.T.Key.TypeParamName == "" {
			encoder, err := p.ifaceKeyEncoder(*n.T.Key)
			if err != nil {
				return "", err
			}
			return "gort$.goEMapFrom([" + joinComma(entries) + "], " + encoder + ")", nil
		}
		return "gort$.goMapFrom([" + joinComma(entries) + "])", nil
	case *ir.MapGet:
		return p.printMapAccess(mapHelper("goMapGet", n.Map), n.Map, n.Key, n.T)
	case *ir.MapLookup:
		return p.printMapAccess(mapHelper("goMapLookup", n.Map), n.Map, n.Key, n.T)
	case *ir.MapLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort$." + mapHelper("goMapLen", n.X) + "(" + x + ")", nil
	case *ir.StringLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort$.goStringLen(" + x + ")", nil
	case *ir.MinMax:
		values, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		if n.Max {
			return "gort$.goMax([" + values + "])", nil
		}
		return "gort$.goMin([" + values + "])", nil
	case *ir.StringConvert:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		switch n.Op {
		case "fromRune":
			return "gort$.goStringFromRune(" + x + ")", nil
		case "fromBytes":
			return "gort$.goStringFromBytes(" + x + ")", nil
		case "fromRunes":
			return "gort$.goStringFromRunes(" + x + ")", nil
		case "toBytes":
			return "gosl$.goSliceFrom(gort$.goStringBytes(" + x + "))", nil
		case "toRunes":
			return "gosl$.goSliceFrom(gort$.goStringRunes(" + x + "))", nil
		}
		return "", fmt.Errorf("no emission for string conversion %q", n.Op)
	case *ir.StringIndex:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		index, err := p.printExpr(n.Index)
		if err != nil {
			return "", err
		}
		return "gort$.goStringIndex(" + x + ", " + index + ")", nil
	case *ir.StringSlice:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		low := "0"
		if n.Low != nil {
			if low, err = p.printExpr(n.Low); err != nil {
				return "", err
			}
		}
		high := "undefined"
		if n.High != nil {
			if high, err = p.printExpr(n.High); err != nil {
				return "", err
			}
		}
		return "gort$.goStringSlice(" + x + ", " + low + ", " + high + ")", nil
	case *ir.SliceLit:
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "gosl$.goSliceFrom([" + values + "])", nil
	case *ir.SliceMake:
		length, err := p.printExpr(n.Length)
		if err != nil {
			return "", err
		}
		// An omitted capacity defaults inside the helper, so the length
		// expression evaluates exactly once.
		capacity := "undefined"
		if n.Capacity != nil {
			capacity, err = p.printExpr(n.Capacity)
			if err != nil {
				return "", err
			}
		}
		zero, err := p.zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		switch n.T.Elem.Kind {
		case ir.KindStruct, ir.KindArray, ir.KindExternal, ir.KindIface:
			// Every element needing independent storage (struct, array,
			// external handle, or a type-parameter carrier) is a distinct
			// fresh zero, never one shared mutable object.
			return "gosl$.goSliceMakeStruct(" + length + ", " + capacity + ", () => " + zero + ")", nil
		}
		return "gosl$.goSliceMake(" + length + ", " + capacity + ", " + zero + ")", nil
	case *ir.SliceGet:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		index, err := p.printExpr(n.Index)
		if err != nil {
			return "", err
		}
		if p.nativeSlice(n.X) {
			return "gosl$.goArrayGet(" + x + ", " + index + ")", nil
		}
		return "gosl$.goSliceGet(" + x + ", " + index + ")", nil
	case *ir.SliceReslice:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		low := "0"
		if n.Low != nil {
			low, err = p.printExpr(n.Low)
			if err != nil {
				return "", err
			}
		}
		// An omitted high bound defaults to len(s) inside the helper,
		// computed from the single evaluation of the operand.
		high := "undefined"
		if n.High != nil {
			high, err = p.printExpr(n.High)
			if err != nil {
				return "", err
			}
		}
		if n.Max != nil {
			// The three-index form caps the result at max-low; high is
			// mandatory here, so it is always the printed bound.
			max, err := p.printExpr(n.Max)
			if err != nil {
				return "", err
			}
			return "gosl$.goSliceSlice3(" + x + ", " + low + ", " + high + ", " + max + ")", nil
		}
		return "gosl$.goSliceSlice(" + x + ", " + low + ", " + high + ")", nil
	case *ir.SliceAppend:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		zero, err := p.zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		if clone := p.paramCloneOf(*n.T.Elem); clone != "" {
			return "gosl$.goSliceAppendWith(" + x + ", [" + values + "], () => " + zero + ", " + clone + ", " + p.paramSetOf(*n.T.Elem) + ")", nil
		}
		if n.T.Elem.Kind == ir.KindStruct {
			return "gosl$.goSliceAppendStruct(" + x + ", [" + values + "], () => " + zero + ")", nil
		}
		return "gosl$.goSliceAppend(" + x + ", [" + values + "], " + zero + ")", nil
	case *ir.SliceAppendSlice:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		source, err := p.printExpr(n.Source)
		if err != nil {
			return "", err
		}
		zero, err := p.zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		if n.T.Elem != nil {
			if clone := p.paramCloneOf(*n.T.Elem); clone != "" {
				return "gosl$.goSliceAppendSliceWith(" + x + ", " + source + ", () => " + zero + ", " + clone + ", " + p.paramSetOf(*n.T.Elem) + ")", nil
			}
		}
		if n.T.Elem != nil && n.T.Elem.Kind == ir.KindStruct {
			return "gosl$.goSliceAppendSliceStruct(" + x + ", " + source + ", () => " + zero + ")", nil
		}
		return "gosl$.goSliceAppendSlice(" + x + ", " + source + ", " + zero + ")", nil
	case *ir.SliceCopy:
		dst, err := p.printExpr(n.Dst)
		if err != nil {
			return "", err
		}
		src, err := p.printExpr(n.Src)
		if err != nil {
			return "", err
		}
		if n.Dst.Type().Elem != nil {
			if clone := p.paramCloneOf(*n.Dst.Type().Elem); clone != "" {
				return "gosl$.goSliceCopyWith(" + dst + ", " + src + ", " + clone + ", " + p.paramSetOf(*n.Dst.Type().Elem) + ")", nil
			}
		}
		if n.Dst.Type().Elem != nil && n.Dst.Type().Elem.Kind == ir.KindStruct {
			return "gosl$.goSliceCopyStruct(" + dst + ", " + src + ")", nil
		}
		return "gosl$.goSliceCopy(" + dst + ", " + src + ")", nil
	case *ir.SliceLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		if p.nativeSlice(n.X) {
			return "gosl$.goArrayLen(" + x + ")", nil
		}
		return "gosl$.goSliceLen(" + x + ")", nil
	case *ir.SliceCap:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl$.goSliceCap(" + x + ")", nil
	}
	return "", errNotCollection
}

// mapHelper selects the plain or keyed carrier family for one map
// operand by its key kind.
func mapHelper(name string, mapExpr ir.Expr) string {
	t := mapExpr.Type()
	if t.Key != nil && t.Key.Kind == ir.KindStruct {
		return "goKMap" + strings.TrimPrefix(name, "goMap")
	}
	if t.Key != nil && t.Key.Kind.Float() {
		return "goFMap" + strings.TrimPrefix(name, "goMap")
	}
	if t.Key != nil && t.Key.Kind == ir.KindIface && t.Key.TypeParamName == "" {
		return "goEMap" + strings.TrimPrefix(name, "goMap")
	}
	return name
}

// ifaceKeyEncoder returns the union $key encoder reference for an
// interface-keyed map's key type (marking the requirement so the encoder
// emits beside the union), or "" for every other key kind.
func (p *printer) ifaceKeyEncoder(keyT ir.Type) (string, error) {
	if keyT.Kind != ir.KindIface || keyT.TypeParamName != "" {
		return "", nil
	}
	name, err := p.ifaceUnionAlias(keyT)
	if err != nil {
		return "", err
	}
	p.module.RequireIfaceKeyFn(name)
	return name + "$key", nil
}

// printMapAccess emits a map read with the exact zero value of the map's
// value type.
func (p *printer) printMapAccess(helper string, mapExpr, key ir.Expr, valueType ir.Type) (string, error) {
	m, err := p.printExpr(mapExpr)
	if err != nil {
		return "", err
	}
	k, err := p.printExpr(key)
	if err != nil {
		return "", err
	}
	zero, err := p.zeroLiteral(valueType)
	if err != nil {
		return "", err
	}
	if mapExpr.Type().Key != nil {
		encoder, err := p.ifaceKeyEncoder(*mapExpr.Type().Key)
		if err != nil {
			return "", err
		}
		if encoder != "" {
			return "gort$." + helper + "(" + m + ", " + k + ", " + zero + ", " + encoder + ")", nil
		}
	}
	return "gort$." + helper + "(" + m + ", " + k + ", " + zero + ")", nil
}
