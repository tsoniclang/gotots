// Interface expression emission: boxing, dynamic dispatch, equality,
// and assertions over the shared rtti carrier.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printIfaceExpr renders the interface-family expression nodes; ok
// reports whether the node was one of them.
func (p *printer) printIfaceExpr(e ir.Expr) (string, bool, error) {
	switch n := e.(type) {
	case *ir.IfaceBox:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		rtti, err := p.rttiRef(n.Rtti)
		if err != nil {
			return "", true, err
		}
		return "goif$.goIfaceBox(" + rtti + ", " + x + ")", true, nil
	case *ir.IfaceCall:
		recv, err := p.printExpr(n.Recv)
		if err != nil {
			return "", true, err
		}
		args, err := p.printArgs(n.Args)
		if err != nil {
			return "", true, err
		}
		call := fmt.Sprintf("goif$.goIfaceCall(%s, %q, [%s])", recv, n.Method, args)
		printed, err := p.castResults(call, n.Results)
		return printed, true, err
	case *ir.TypeAssert:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		rtti, err := p.rttiRef(n.Rtti)
		if err != nil {
			return "", true, err
		}
		if n.CommaOk {
			zero, err := p.zeroLiteral(n.Target)
			if err != nil {
				return "", true, err
			}
			return "goif$.goIfaceLookup(" + x + ", " + rtti + ", " + zero + ")", true, nil
		}
		spelled, err := p.tsType(n.Target)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("(goif$.goIfaceAssert(%s, %s, %q) as (%s))", x, rtti, n.SourceDisplay, spelled), true, nil
	case *ir.StructEqual:
		left, err := p.printExpr(n.L)
		if err != nil {
			return "", true, err
		}
		right, err := p.printExpr(n.R)
		if err != nil {
			return "", true, err
		}
		op := "==="
		if n.Negate {
			op = "!=="
		}
		return "(" + left + ".goKey$() " + op + " " + right + ".goKey$())", true, nil
	case *ir.IfaceEqual:
		left, err := p.printExpr(n.L)
		if err != nil {
			return "", true, err
		}
		right, err := p.printExpr(n.R)
		if err != nil {
			return "", true, err
		}
		call := "goif$.goIfaceEqual(" + left + ", " + right + ")"
		if n.Negate {
			return "(!" + call + ")", true, nil
		}
		return call, true, nil
	case *ir.IfaceConcreteEqual:
		iface, err := p.printExpr(n.Iface)
		if err != nil {
			return "", true, err
		}
		concrete, err := p.printExpr(n.Concrete)
		if err != nil {
			return "", true, err
		}
		rtti, err := p.rttiRef(n.Rtti)
		if err != nil {
			return "", true, err
		}
		helper := "goIfaceEqualPrim"
		if n.Form == "key" {
			helper = "goIfaceEqualKey"
		}
		var call string
		if n.IfaceLeft {
			call = fmt.Sprintf("goif$.%s(%s, %s, %s)", helper, iface, rtti, concrete)
		} else {
			// The concrete operand is on the left: the staged arrow keeps
			// Go's left-to-right evaluation.
			call = fmt.Sprintf("(($c) => goif$.%s(%s, %s, $c))(%s)", helper, iface, rtti, concrete)
		}
		if n.Negate {
			return "(!" + call + ")", true, nil
		}
		return call, true, nil
	case *ir.IfaceAssert:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		methods := make([]string, len(n.Methods))
		for i, method := range n.Methods {
			methods[i] = fmt.Sprintf("%q", method)
		}
		list := "[" + joinComma(methods) + "]"
		if n.CommaOk {
			return fmt.Sprintf("goif$.goIfaceLookupIface(%s, %s)", x, list), true, nil
		}
		spelled, err := p.tsType(n.Target)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("(goif$.goIfaceAssertIface(%s, %s, %q, %q) as (%s))", x, list, n.SourceDisplay, n.TargetDisplay, spelled), true, nil
	}
	return "", false, nil
}
