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
		if n.Type().Key != nil && n.Type().Key.Kind == ir.KindStruct {
			return "gort$.goKMapMake()", nil
		}
		return "gort$.goMapMake()", nil
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
	return name
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
	return "gort$." + helper + "(" + m + ", " + k + ", " + zero + ")", nil
}
