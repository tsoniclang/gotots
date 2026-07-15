package emit

import (
	"fmt"
	"go/token"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/ir"
)

func printConst(n *ir.Const) (string, error) {
	kind := n.T.Kind
	switch {
	case kind == ir.KindBool, kind == ir.KindString, kind.Float(), kind == ir.KindUnit:
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
	case from == to && (to == ir.KindString || to == ir.KindBool || to == ir.KindFloat64):
		// Same-carrier conversion between named types: identity.
		return "(" + x + ")", nil

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
