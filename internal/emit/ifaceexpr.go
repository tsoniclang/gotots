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
		vtable, err := p.boxVtable(n.Rtti)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("goif$.goIfaceBox(%q, %s, %s, %s)", boxDiscriminant(n.Rtti), rtti, x, vtable), true, nil
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
		tokens := make([]string, 0, len(n.Implementers))
		for _, token := range n.Implementers {
			// A withheld package's token cannot exist at runtime (its
			// module never evaluates) and must not be referenced.
			if token.Pkg != "" && p.module.Withheld != nil && p.module.Withheld(token.Pkg) {
				continue
			}
			spelled, err := p.rttiRef(token)
			if err != nil {
				return "", true, err
			}
			tokens = append(tokens, spelled)
		}
		list := "[" + joinComma(tokens) + "]"
		required := make([]string, len(n.Required))
		for i, name := range n.Required {
			required[i] = fmt.Sprintf("%q", name)
		}
		reqList := "[" + joinComma(required) + "]"
		if n.CommaOk {
			targetUnion, err := p.tsType(n.Target)
			if err != nil {
				return "", true, err
			}
			return fmt.Sprintf("goif$.goIfaceLookupSet<Exclude<%s, undefined>>(%s, %s)", targetUnion, x, list), true, nil
		}
		spelled, err := p.tsType(n.Target)
		if err != nil {
			return "", true, err
		}
		return fmt.Sprintf("goif$.goIfaceAssertSet<Exclude<%s, undefined>>(%s, %s, %s, %q, %q)", spelled, x, list, reqList, n.SourceDisplay, n.TargetDisplay), true, nil
	}
	return "", false, nil
}

// printIfaceCall lowers an interface method call onto the closed
// discriminated union (ADR-0004): the literal switch narrows each member
// so the payload and vtable are exactly typed — no cast, no erased
// recovery, no value import of any implementer. A nil interface panics
// first, exactly Go.
func (p *printer) printIfaceCall(n *ir.IfaceCall) (string, error) {
	recv, err := p.printExpr(n.Recv)
	if err != nil {
		return "", err
	}
	recvType := n.Recv.Type()
	union, err := p.tsType(recvType)
	if err != nil {
		return "", err
	}
	params := []string{"$r: " + union}
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
	if len(p.retainedMembers(recvType)) == 0 {
		// Every implementer is withheld: no value of this interface can
		// exist at runtime in this bundle, so the call is unreachable.
		// Routing through goIndirect keeps TypeScript from applying IIFE
		// control-flow analysis (which would mark the CALLER's subsequent
		// statements unreachable and disable narrowing there).
		sub.line("gort$.goPanicUnreachableType(%q);", recvType.Go)
		closing := strings.Repeat("  ", p.indent)
		return fmt.Sprintf("(gort$.goIndirect((%s): %s => {\n%s%s}))(%s)",
			joinComma(params), result, body.String(), closing, joinComma(passed)), nil
	}
	sub.line("switch ($r.k) {")
	sub.indent++
	for _, member := range p.retainedMembers(recvType) {
		call := "$r.m." + n.Display + "(" + joinComma(append([]string{"$r.v"}, argNames...)) + ")"
		if result == "void" {
			sub.line("case %q: %s; return;", member.K, call)
		} else {
			sub.line("case %q: return %s;", member.K, call)
		}
	}
	sub.line("default: gort$.goPanicUnreachableType(($r as goif$.GoAnyBox).r.d);")
	sub.indent--
	sub.line("}")
	closing := strings.Repeat("  ", p.indent)
	return fmt.Sprintf("((%s): %s => {\n%s%s})(%s)",
		joinComma(params), result, body.String(), closing, joinComma(passed)), nil
}
