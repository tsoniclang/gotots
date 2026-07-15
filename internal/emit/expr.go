package emit

import (
	"fmt"
	"go/token"
	"strings"

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
func (p *printer) printExpr(e ir.Expr) (string, error) {
	switch n := e.(type) {
	case *ir.Const:
		return printConst(n)
	case *ir.VarRef:
		return n.Name, nil
	case *ir.Binary:
		return p.printBinary(n)
	case *ir.Unary:
		return p.printUnary(n)
	case *ir.Convert:
		return p.printConvert(n)
	case *ir.Call:
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.Callee)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s(%s)", callee, args), nil
	case *ir.MethodCall:
		recv, err := p.printExpr(n.Recv)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.TypeName+"$"+n.Method)
		if err != nil {
			return "", err
		}
		// A value receiver dereferences a pointer caller at the call (Go
		// copies the pointee); a pointer receiver takes the pointer as
		// is — nil receivers run, as in Go.
		if !n.PointerRecv && n.Recv.Type().Kind == ir.KindPointer {
			recv = "gort.goNilCheck(" + recv + ")"
		}
		if args == "" {
			return fmt.Sprintf("%s(%s)", callee, recv), nil
		}
		return fmt.Sprintf("%s(%s, %s)", callee, recv, args), nil
	case *ir.FieldLoad:
		base, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		if n.X.Type().Kind == ir.KindStruct {
			return fmt.Sprintf("%s.%s", base, n.Field), nil
		}
		return fmt.Sprintf("gort.goNilCheck(%s).%s", base, n.Field), nil
	case *ir.StructNew:
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		class, err := p.module.symbol(n.Pkg, n.TypeName)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("new %s(%s)", class, args), nil
	case *ir.Closure:
		return p.printClosure(n)
	case *ir.FuncRef:
		return p.module.symbol(n.Pkg, n.Name)
	case *ir.DynCall:
		fun, err := p.printExpr(n.Fun)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("gort.goNilCheck(%s)(%s)", fun, args), nil
	case *ir.StructCopy:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return x + ".goClone$()", nil
	case *ir.StructZero:
		return p.zeroLiteral(n.T)
	case *ir.AddrOf:
		// The pointer to an addressable struct value is the instance.
		return p.printExpr(n.X)
	case *ir.Deref:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort.goNilCheck(" + x + ")", nil
	case *ir.NilConst:
		return "undefined", nil
	case *ir.IsNil:
		x, err := p.printExpr(n.X)
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
		return "gort.goMapFrom([" + joinComma(entries) + "])", nil
	case *ir.MapGet:
		return p.printMapAccess("goMapGet", n.Map, n.Key, n.T)
	case *ir.MapLookup:
		return p.printMapAccess("goMapLookup", n.Map, n.Key, n.T)
	case *ir.MapLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort.goMapLen(" + x + ")", nil
	case *ir.StringLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gort.goStringLen(" + x + ")", nil
	case *ir.SliceLit:
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceFrom([" + values + "])", nil
	case *ir.SliceMake:
		length, err := p.printExpr(n.Length)
		if err != nil {
			return "", err
		}
		capacity := length
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
		if n.T.Elem.Kind == ir.KindStruct {
			// Every struct element is a distinct fresh zero instance.
			return "gosl.goSliceMakeStruct(" + length + ", " + capacity + ", () => " + zero + ")", nil
		}
		return "gosl.goSliceMake(" + length + ", " + capacity + ", " + zero + ")", nil
	case *ir.SliceGet:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		index, err := p.printExpr(n.Index)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceGet(" + x + ", " + index + ")", nil
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
		high := "gosl.goSliceLen(" + x + ")"
		if n.High != nil {
			high, err = p.printExpr(n.High)
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
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		values, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceAppend(" + x + ", [" + values + "])", nil
	case *ir.SliceLen:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceLen(" + x + ")", nil
	case *ir.SliceCap:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		return "gosl.goSliceCap(" + x + ")", nil
	}
	return "", fmt.Errorf("no emission for IR expression %T", e)
}

// printClosure emits a function literal as an arrow function: JS arrows
// capture enclosing variables by reference with per-iteration loop
// bindings, exactly matching Go's capture semantics.
func (p *printer) printClosure(n *ir.Closure) (string, error) {
	var params []string
	for _, parameter := range n.Params {
		spelled, err := p.tsType(parameter.Type)
		if err != nil {
			return "", err
		}
		params = append(params, parameter.Name+": "+spelled)
	}
	result, err := p.tsResultType(n.Results)
	if err != nil {
		return "", err
	}
	var sub strings.Builder
	subPrinter := &printer{out: &sub, module: p.module, indent: p.indent + 1}
	if err := subPrinter.printBlockBody(n.Body); err != nil {
		return "", err
	}
	closing := strings.Repeat("  ", p.indent)
	return "((" + strings.Join(params, ", ") + "): " + result + " => {\n" + sub.String() + closing + "})", nil
}

func (p *printer) printArgs(args []ir.Expr) (string, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		printed, err := p.printExpr(arg)
		if err != nil {
			return "", err
		}
		parts[i] = printed
	}
	return joinComma(parts), nil
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
	return "gort." + helper + "(" + m + ", " + k + ", " + zero + ")", nil
}

// zeroLiteral spells the Go zero value of a reviewed type in TypeScript.
// A struct zero is a fresh instance from the class's zero factory.
func (p *printer) zeroLiteral(t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindBool:
		return "false", nil
	case t.Kind == ir.KindString:
		return `""`, nil
	case t.Kind.Nilable():
		return "undefined", nil
	case t.Kind == ir.KindStruct:
		class, err := p.module.symbol(t.Pkg, t.Named)
		if err != nil {
			return "", err
		}
		return class + ".goZero$()", nil
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

func (p *printer) printBinary(n *ir.Binary) (string, error) {
	left, err := p.printExpr(n.L)
	if err != nil {
		return "", err
	}
	right, err := p.printExpr(n.R)
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

func (p *printer) printUnary(n *ir.Unary) (string, error) {
	x, err := p.printExpr(n.X)
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

func (p *printer) printConvert(n *ir.Convert) (string, error) {
	x, err := p.printExpr(n.X)
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
