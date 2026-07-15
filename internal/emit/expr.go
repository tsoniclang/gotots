package emit

import (
	"fmt"
	"go/token"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/ir"
)

// family maps an IR integer kind to its ABI carrier family.
func family(kind ir.Kind) (abi.Family, bool) {
	switch kind {
	case ir.KindInt8, ir.KindInt16, ir.KindInt32:
		return abi.FamilyNumberSigned, true
	case ir.KindUint8, ir.KindUint16, ir.KindUint32:
		return abi.FamilyNumberUnsigned, true
	case ir.KindInt64, ir.KindInt:
		return abi.FamilyBigSigned, true
	case ir.KindUint64, ir.KindUint, ir.KindUintptr:
		return abi.FamilyBigUnsigned, true
	}
	return "", false
}

func helper(name string) string { return "goabi." + name }

// printExpr renders one IR expression, fully parenthesized.
func printExpr(e ir.Expr) (string, error) {
	switch n := e.(type) {
	case *ir.Const:
		return printConst(n)
	case *ir.VarRef:
		return n.Name, nil
	case *ir.Binary:
		return printBinary(n)
	case *ir.Unary:
		return printUnary(n)
	case *ir.Convert:
		return printConvert(n)
	case *ir.Call:
		args, err := printArgs(n.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s)", n.Callee, args), nil
	case *ir.MethodCall:
		recv, err := printExpr(n.Recv)
		if err != nil {
			return "", err
		}
		args, err := printArgs(n.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("gort.goNilCheck(%s).%s(%s)", recv, n.Method, args), nil
	case *ir.FieldLoad:
		base, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("gort.goNilCheck(%s).%s", base, n.Field), nil
	case *ir.StructNew:
		args, err := printArgs(n.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("new %s(%s)", n.TypeName, args), nil
	case *ir.NilConst:
		return "undefined", nil
	case *ir.IsNil:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		if n.Negate {
			return "(" + x + " !== undefined)", nil
		}
		return "(" + x + " === undefined)", nil
	case *ir.MapMake:
		return "gort.goMapMake()", nil
	case *ir.MapFrom:
		var entries []string
		for i := range n.Keys {
			key, err := printExpr(n.Keys[i])
			if err != nil {
				return "", err
			}
			value, err := printExpr(n.Values[i])
			if err != nil {
				return "", err
			}
			entries = append(entries, "["+key+", "+value+"]")
		}
		return "gort.goMapFrom([" + joinComma(entries) + "])", nil
	case *ir.MapGet:
		return printMapAccess("goMapGet", n.Map, n.Key, n.T)
	case *ir.MapLookup:
		return printMapAccess("goMapLookup", n.Map, n.Key, n.T)
	case *ir.MapLen:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort.goMapLen(" + x + ")", nil
	case *ir.StringLen:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort.goStringLen(" + x + ")", nil
	case *ir.SliceLit:
		values, err := printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceFrom([" + values + "])", nil
	case *ir.SliceMake:
		length, err := printExpr(n.Length)
		if err != nil {
			return "", err
		}
		capacity := length
		if n.Capacity != nil {
			capacity, err = printExpr(n.Capacity)
			if err != nil {
				return "", err
			}
		}
		zero, err := zeroLiteral(*n.T.Elem)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceMake(" + length + ", " + capacity + ", " + zero + ")", nil
	case *ir.SliceGet:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		index, err := printExpr(n.Index)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceGet(" + x + ", " + index + ")", nil
	case *ir.SliceReslice:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		low := "0"
		if n.Low != nil {
			low, err = printExpr(n.Low)
			if err != nil {
				return "", err
			}
		}
		high := "gosl.goSliceLen(" + x + ")"
		if n.High != nil {
			high, err = printExpr(n.High)
			if err != nil {
				return "", err
			}
		}
		// The operand is evaluated once even when the default high bound
		// re-reads its length; a temp is unnecessary for pure operands and
		// the builder only produces pure slice operands today (vars,
		// fields of pure bases). Revisit with effectful operands.
		return "gosl.goSliceSlice(" + x + ", " + low + ", " + high + ")", nil
	case *ir.SliceAppend:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		values, err := printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceAppend(" + x + ", [" + values + "])", nil
	case *ir.SliceLen:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceLen(" + x + ")", nil
	case *ir.SliceCap:
		x, err := printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceCap(" + x + ")", nil
	}
	return "", fmt.Errorf("no emission for IR expression %T", e)
}

func printArgs(args []ir.Expr) (string, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		printed, err := printExpr(arg)
		if err != nil {
			return "", err
		}
		parts[i] = printed
	}
	return joinComma(parts), nil
}

// printMapAccess emits a map read with the exact zero value of the map's
// value type.
func printMapAccess(helper string, mapExpr, key ir.Expr, valueType ir.Type) (string, error) {
	m, err := printExpr(mapExpr)
	if err != nil {
		return "", err
	}
	k, err := printExpr(key)
	if err != nil {
		return "", err
	}
	zero, err := zeroLiteral(valueType)
	if err != nil {
		return "", err
	}
	return "gort." + helper + "(" + m + ", " + k + ", " + zero + ")", nil
}

// zeroLiteral spells the Go zero value of a reviewed type in TypeScript.
func zeroLiteral(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindBool:
		return "false", nil
	case t.Kind == ir.KindString:
		return `""`, nil
	case t.Kind == ir.KindPointer, t.Kind == ir.KindMap, t.Kind == ir.KindSlice:
		return "undefined", nil
	case t.Kind.Wide64():
		return "0n", nil
	case t.Kind.Integer(), t.Kind.Float():
		return "0", nil
	}
	return "", fmt.Errorf("no zero literal for type %q", t.Go)
}

func printConst(n *ir.Const) (string, error) {
	kind := n.T.Kind
	switch {
	case kind == ir.KindBool, kind == ir.KindString, kind.Float():
		return n.Value, nil
	case kind.Wide64():
		return "(" + n.Value + "n)", nil
	case kind.Integer():
		return "(" + n.Value + ")", nil
	}
	return "", fmt.Errorf("no emission for constant of type %q", n.T.Go)
}

func printBinary(n *ir.Binary) (string, error) {
	left, err := printExpr(n.L)
	if err != nil {
		return "", err
	}
	right, err := printExpr(n.R)
	if err != nil {
		return "", err
	}
	operand := n.L.Type().Kind

	// Comparisons and boolean logic are direct for every reviewed carrier:
	// canonical-range numbers, bigints, booleans, and string equality.
	switch n.Op {
	case token.EQL:
		return "(" + left + " === " + right + ")", nil
	case token.NEQ:
		return "(" + left + " !== " + right + ")", nil
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		return "(" + left + " " + n.Op.String() + " " + right + ")", nil
	case token.LAND:
		return "(" + left + " && " + right + ")", nil
	case token.LOR:
		return "(" + left + " || " + right + ")", nil
	}

	// String concatenation and float64 arithmetic are exact directly.
	if operand == ir.KindString && n.Op == token.ADD {
		return "(" + left + " + " + right + ")", nil
	}
	if operand == ir.KindFloat64 {
		switch n.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO:
			return "(" + left + " " + n.Op.String() + " " + right + ")", nil
		}
		return "", fmt.Errorf("no emission for float64 operator %s", n.Op)
	}

	carrierFamily, isInteger := family(operand)
	if !isInteger {
		return "", fmt.Errorf("no emission for operator %s on %q", n.Op, n.L.Type().Go)
	}
	bits := operand.Bits()
	wrap := helper(abi.Wrap(carrierFamily, bits))
	operation := func(name string) string { return helper(abi.Op(carrierFamily, bits, name)) }
	isBig := operand.Wide64()

	switch n.Op {
	case token.ADD, token.SUB:
		if isBig {
			name := "add"
			if n.Op == token.SUB {
				name = "sub"
			}
			return operation(name) + "(" + left + ", " + right + ")", nil
		}
		return wrap + "(" + left + " " + n.Op.String() + " " + right + ")", nil
	case token.MUL:
		return operation("mul") + "(" + left + ", " + right + ")", nil
	case token.QUO:
		return operation("div") + "(" + left + ", " + right + ")", nil
	case token.REM:
		return operation("rem") + "(" + left + ", " + right + ")", nil
	case token.AND, token.OR, token.XOR, token.AND_NOT:
		if isBig {
			name := map[token.Token]string{token.AND: "and", token.OR: "or", token.XOR: "xor", token.AND_NOT: "andNot"}[n.Op]
			return operation(name) + "(" + left + ", " + right + ")", nil
		}
		js := map[token.Token]string{token.AND: "&", token.OR: "|", token.XOR: "^"}[n.Op]
		if n.Op == token.AND_NOT {
			return wrap + "(" + left + " & ~" + right + ")", nil
		}
		return wrap + "(" + left + " " + js + " " + right + ")", nil
	case token.SHL, token.SHR:
		count, err := shiftCount(n.R, right)
		if err != nil {
			return "", err
		}
		name := "shl"
		if n.Op == token.SHR {
			name = "shr"
		}
		return operation(name) + "(" + left + ", " + count + ")", nil
	}
	return "", fmt.Errorf("no emission for integer operator %s", n.Op)
}

// shiftCount adapts the count operand to the ABI's number-typed count with
// the exact Go negative-count panic and beyond-width clamping.
func shiftCount(countExpr ir.Expr, printed string) (string, error) {
	kind := countExpr.Type().Kind
	if !kind.Integer() {
		return "", fmt.Errorf("shift count of type %q", countExpr.Type().Go)
	}
	if kind.Wide64() {
		return helper("goShiftCountFromBig") + "(" + printed + ")", nil
	}
	return printed, nil
}

func printUnary(n *ir.Unary) (string, error) {
	x, err := printExpr(n.X)
	if err != nil {
		return "", err
	}
	kind := n.T.Kind
	switch n.Op {
	case token.NOT:
		return "(!" + x + ")", nil
	case token.SUB:
		if kind == ir.KindFloat64 {
			return "(-" + x + ")", nil
		}
		carrierFamily, ok := family(kind)
		if !ok {
			return "", fmt.Errorf("no emission for negation of %q", n.T.Go)
		}
		if kind.Wide64() {
			return helper(abi.Op(carrierFamily, kind.Bits(), "neg")) + "(" + x + ")", nil
		}
		return helper(abi.Wrap(carrierFamily, kind.Bits())) + "(-" + x + ")", nil
	case token.XOR:
		carrierFamily, ok := family(kind)
		if !ok {
			return "", fmt.Errorf("no emission for complement of %q", n.T.Go)
		}
		if kind.Wide64() {
			return helper(abi.Op(carrierFamily, kind.Bits(), "not")) + "(" + x + ")", nil
		}
		return helper(abi.Wrap(carrierFamily, kind.Bits())) + "(~" + x + ")", nil
	}
	return "", fmt.Errorf("no emission for unary operator %s", n.Op)
}

func printConvert(n *ir.Convert) (string, error) {
	x, err := printExpr(n.X)
	if err != nil {
		return "", err
	}
	from := n.X.Type().Kind
	to := n.To.Kind

	switch {
	case to == ir.KindFloat64 && from.Integer():
		if from.Wide64() {
			return "Number(" + x + ")", nil
		}
		return "(" + x + ")", nil

	case to.Integer() && from.Integer():
		toFamily, _ := family(to)
		switch {
		case to.Wide64() && from.Wide64():
			return helper(abi.Wrap(toFamily, 64)) + "(" + x + ")", nil
		case to.Wide64():
			if to == ir.KindUint64 || to == ir.KindUint || to == ir.KindUintptr {
				return helper("goUint64FromNumber") + "(" + x + ")", nil
			}
			return helper("goInt64FromNumber") + "(" + x + ")", nil
		case from.Wide64():
			name := map[ir.Kind]string{
				ir.KindInt8: "goInt8FromBig", ir.KindInt16: "goInt16FromBig", ir.KindInt32: "goInt32FromBig",
				ir.KindUint8: "goUint8FromBig", ir.KindUint16: "goUint16FromBig", ir.KindUint32: "goUint32FromBig",
			}[to]
			if name == "" {
				return "", fmt.Errorf("no conversion emission to %q", n.To.Go)
			}
			return helper(name) + "(" + x + ")", nil
		default:
			return helper(abi.Wrap(toFamily, to.Bits())) + "(" + x + ")", nil
		}
	}
	return "", fmt.Errorf("no conversion emission from %q to %q", n.X.Type().Go, n.To.Go)
}
