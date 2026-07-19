// External-bridge and instantiated-generic value printers: the typed
// stub-obligation expressions and eta-expanded generic function values.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printExternBridgeExpr prints the external-contract bridge expressions
// and eta-expanded generic function values.
func (p *printer) printExternBridgeExpr(e ir.Expr) (string, error) {
	switch n := e.(type) {
	case *ir.GenericFuncValue:
		calleeName := n.Name
		if callFamilyEnc(n.TypeArgs, n.HardKeyed, p.familyEnc) {
			calleeName += "$ek"
		}
		callee, err := p.module.symbol(n.Pkg, calleeName)
		if err != nil {
			return "", err
		}
		typeArgs := make([]string, len(n.TypeArgs))
		for i, arg := range n.TypeArgs {
			if typeArgs[i], err = p.tsType(arg); err != nil {
				return "", err
			}
		}
		factories, err := p.zeroFactoryArgs(n.TypeArgs, n.KeyedParams)
		if err != nil {
			return "", err
		}
		params := make([]string, 0, len(n.T.Sig.Params))
		args := make([]string, 0, len(n.T.Sig.Params))
		for i, param := range n.T.Sig.Params {
			spelled, err := p.tsType(param)
			if err != nil {
				return "", err
			}
			params = append(params, fmt.Sprintf("$a%d: %s", i, spelled))
			args = append(args, fmt.Sprintf("$a%d", i))
		}
		if factories != "" {
			args = append(args, factories)
		}
		// Eta-expansion: the instantiated generic as an exactly typed
		// arrow closing over the instantiation's factory derivations.
		return "((" + strings.Join(params, ", ") + ") => " + callee + "<" + strings.Join(typeArgs, ", ") + ">(" + strings.Join(args, ", ") + "))", nil
	case *ir.ExternEqual:
		left, err := p.printExpr(n.L)
		if err != nil {
			return "", err
		}
		right, err := p.printExpr(n.R)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.TypeName+"$eq$")
		if err != nil {
			return "", err
		}
		call := callee + "(" + left + ", " + right + ")"
		if n.Negate {
			return "(!" + call + ")", nil
		}
		return call, nil
	case *ir.ExternFieldRead:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		callee, err := p.module.symbol(n.Pkg, n.Symbol)
		if err != nil {
			return "", err
		}
		return callee + "(" + x + ")", nil
	case *ir.ExternToOwned:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", err
		}
		spelledX, err := p.tsType(n.X.Type())
		if err != nil {
			return "", err
		}
		class, err := p.module.symbol(n.To.Pkg, n.To.Named)
		if err != nil {
			return "", err
		}
		args := make([]string, 0, len(n.FieldSymbols))
		for _, symbol := range n.FieldSymbols {
			callee, err := p.module.symbol(n.X.Type().Pkg, symbol)
			if err != nil {
				return "", err
			}
			args = append(args, callee+"($x)")
		}
		// The operand evaluates ONCE (Go's conversion evaluation).
		return "(($x: " + spelledX + ") => new " + class + "(" + joinComma(args) + "))(" + x + ")", nil
	case *ir.ExternLit:
		callee, err := p.module.symbol(n.T.Pkg, n.Symbol)
		if err != nil {
			return "", err
		}
		args, err := p.printArgs(n.Values)
		if err != nil {
			return "", err
		}
		return callee + "(" + args + ")", nil
	}
	return "", fmt.Errorf("unhandled extern-bridge expression %T", e)
}
