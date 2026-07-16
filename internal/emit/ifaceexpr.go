// Interface expression emission: boxing, dynamic dispatch, equality,
// and assertions over the shared rtti carrier.
package emit

import (
	"fmt"
	"strings"

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
		printed, err := p.printIfaceCall(n)
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
		call := left + ".goEq$(" + right + ")"
		if n.Negate {
			return "(!" + call + ")", true, nil
		}
		return call, true, nil
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
		if n.Form == "via" {
			helper = "goIfaceEqualVia"
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
		tokens := make([]string, len(n.Implementers))
		for i, token := range n.Implementers {
			spelled, err := p.rttiRef(token)
			if err != nil {
				return "", true, err
			}
			tokens[i] = spelled
		}
		list := "[" + joinComma(tokens) + "]"
		required := make([]string, len(n.Required))
		for i, name := range n.Required {
			required[i] = fmt.Sprintf("%q", name)
		}
		reqList := "[" + joinComma(required) + "]"
		if n.CommaOk {
			return fmt.Sprintf("goif$.goIfaceLookupSet(%s, %s)", x, list), true, nil
		}
		spelled, err := p.tsType(n.Target)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("(goif$.goIfaceAssertSet(%s, %s, %s, %q, %q) as (%s))", x, list, reqList, n.SourceDisplay, n.TargetDisplay, spelled), true, nil
	}
	return "", false, nil
}

// printIfaceCall lowers an interface method call to an exhaustive token
// switch over the closed dynamic-type set: the receiver and arguments
// evaluate once, a nil interface panics, and each branch dispatches
// directly to the concrete generated method. No name-selected member
// lookup occurs.
func (p *printer) printIfaceCall(n *ir.IfaceCall) (string, error) {
	recv, err := p.printExpr(n.Recv)
	if err != nil {
		return "", err
	}
	params := []string{"$r: goif$.GoIface"}
	passed := []string{recv}
	argNames := make([]string, len(n.Args))
	for i, arg := range n.Args {
		printed, err := p.printExpr(arg)
		if err != nil {
			return "", err
		}
		spelled, err := p.tsType(arg.Type())
		if err != nil {
			return "", err
		}
		name := fmt.Sprintf("$a%d", i)
		params = append(params, name+": "+spelled)
		passed = append(passed, printed)
		argNames[i] = name
	}
	result, err := p.tsFuncResultType(n.Results)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	sub := &printer{out: &body, module: p.module, indent: 0}
	sub.line("if ($r === undefined) { gort$.goPanicNil(); }")
	sub.line("const $box = $r as goif$.GoIfaceBox;")
	sub.line("switch ($box.r) {")
	sub.indent++
	for _, branch := range n.Branches {
		token, err := p.rttiRef(branch.Rtti)
		if err != nil {
			return "", err
		}
		callee, err := p.ifaceBranchCallee(branch)
		if err != nil {
			return "", err
		}
		payload, err := p.tsType(branch.Payload)
		if err != nil {
			return "", err
		}
		receiver := "($box.v as (" + payload + "))"
		for _, field := range branch.FieldPath {
			// A promoted method delegates through the embedded value
			// fields to the declaring type.
			receiver += "." + field
		}
		operands := append([]string{receiver}, argNames...)
		call := callee + "(" + joinComma(operands) + ")"
		if result == "void" {
			sub.line("case %s: %s; return;", token, call)
		} else {
			sub.line("case %s: return %s;", token, call)
		}
	}
	sub.line("default: gort$.goPanicUnreachableType($box.r.d);")
	sub.indent--
	sub.line("}")
	closing := strings.Repeat("  ", p.indent)
	return fmt.Sprintf("((%s): %s => {\n%s%s})(%s)",
		strings.Join(params, ", "), result, body.String(), closing, joinComma(passed)), nil
}

// ifaceBranchCallee spells the direct method function for one dispatch
// branch: the generated DeclType$Method (owned or external stub) of the
// method's declaring type.
func (p *printer) ifaceBranchCallee(branch ir.IfaceBranch) (string, error) {
	if branch.External {
		return p.module.symbol(branch.DeclPkg, externMethodSymbol(branch.DeclType, branch.Method))
	}
	return p.module.symbol(branch.DeclPkg, branch.DeclType+"$"+branch.Method)
}
