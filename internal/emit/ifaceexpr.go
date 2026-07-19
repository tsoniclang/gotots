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
		if n.X.Type().Kind == ir.KindUnit {
			// The unit payload keeps its literal type through inference.
			x = "(0 as 0)"
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
	case *ir.ParamIfaceBox:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		op, has := p.rttiOps[n.Param]
		if !has {
			return "", true, fmt.Errorf("no rt$ operation in scope for parameter %s", n.Param)
		}
		spelled, err := p.tsType(n.T)
		if err != nil {
			return "", true, err
		}
		// The box composes from the binding's triple; its static type is
		// generic, so the union membership the binding evidence guarantees
		// is asserted through unknown.
		return "(goif$.goIfaceBox(" + op + ".k, " + op + ".r, " + x + ", " + op + ".m) as unknown as " + spelled + ")", true, nil
	case *ir.IfaceCall:
		printed, err := p.printIfaceCall(n)
		return printed, true, err
	case *ir.TypeAssert:
		x, err := p.printExpr(n.X)
		if err != nil {
			return "", true, err
		}
		union, err := p.tsType(n.X.Type())
		if err != nil {
			return "", true, err
		}
		k := boxDiscriminant(n.Rtti)
		possible := p.assertTokenPossible(n.X.Type(), k)
		if n.CommaOk {
			zero, err := p.zeroLiteral(n.Target)
			if err != nil {
				return "", true, err
			}
			spelled, err := p.tsType(n.Target)
			if err != nil {
				return "", true, err
			}
			if !possible {
				// The target's token is not a member of the operand's
				// closed union (never boxed in the unit): the assertion is
				// statically false — the operand still evaluates.
				return fmt.Sprintf("(($i: %s): readonly [%s, boolean] => { void $i; return [%s, false]; })(%s)",
					union, spelled, zero, x), true, nil
			}
			// Literal-discriminant narrowing: the true branch reads the
			// member's EXACT payload — no cast, no helper.
			return fmt.Sprintf("(($i: %s): readonly [%s, boolean] => ($i !== undefined && $i.k === %q ? [$i.v, true] : [%s, false]))(%s)",
				union, spelled, k, zero, x), true, nil
		}
		result, err := p.tsType(n.Target)
		if err != nil {
			return "", true, err
		}
		if !possible {
			return fmt.Sprintf("(($i: %s): %s => { gort$.goPanicConversion($i === undefined ? %q : $i.r.d, %q, %q); })(%s)",
				union, result, "nil", n.SourceDisplay, n.TargetDisplay, x), true, nil
		}
		return fmt.Sprintf("(($i: %s): %s => { if ($i !== undefined && $i.k === %q) { return $i.v; } gort$.goPanicConversion($i === undefined ? %q : $i.r.d, %q, %q); })(%s)",
			union, result, k, "nil", n.SourceDisplay, n.TargetDisplay, x), true, nil
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
		union, err := p.tsType(n.Iface)
		if err != nil {
			return "", true, err
		}
		call := union + "$eq(" + left + ", " + right + ")"
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
		ifaceType, err := p.tsType(n.Iface.Type())
		if err != nil {
			return "", true, err
		}
		concreteType, err := p.tsType(n.Concrete.Type())
		if err != nil {
			return "", true, err
		}
		// Inline narrowing, no erased helper: the interface value equals the
		// concrete when its dynamic type IS the concrete type (its literal
		// discriminant) and the exact narrowed payload compares — === for a
		// primitive/pointer carrier, the type's own goEq$ for a comparable
		// value struct. The staged arrow keeps Go's left-to-right order.
		k := boxDiscriminant(n.Rtti)
		valCmp := "$i.v === $c"
		if n.Form == "via" {
			valCmp = "$i.v.goEq$($c)"
		}
		cond := fmt.Sprintf("$i !== undefined && $i.k === %q && (%s)", k, valCmp)
		var call string
		if n.IfaceLeft {
			call = fmt.Sprintf("(($i: %s, $c: %s) => %s)(%s, %s)", ifaceType, concreteType, cond, iface, concrete)
		} else {
			call = fmt.Sprintf("(($c: %s, $i: %s) => %s)(%s, %s)", concreteType, ifaceType, cond, concrete, iface)
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
		union, err := p.tsType(n.X.Type())
		if err != nil {
			return "", true, err
		}
		targetUnion, err := p.tsType(n.Target)
		if err != nil {
			return "", true, err
		}
		// Membership by literal-discriminant ||-chain: TypeScript narrows
		// the source union to exactly the target's member subset — the
		// result needs no cast (its type IS the target union).
		var condition string
		if n.Universal {
			condition = "$i !== undefined"
		} else {
			clauses := make([]string, 0, len(n.Implementers))
			for _, token := range n.Implementers {
				if token.Pkg != "" && p.module.Withheld != nil && p.module.Withheld(token.Pkg) {
					continue
				}
				clauses = append(clauses, fmt.Sprintf("$i.k === %q", boxDiscriminant(token)))
			}
			if len(clauses) == 0 {
				condition = "false"
			} else {
				condition = "$i !== undefined && (" + strings.Join(clauses, " || ") + ")"
			}
		}
		if condition == "false" {
			// No retained implementer: the assertion can never succeed —
			// the true branch would be dead yet still typechecked, so it
			// is not emitted at all.
			if n.CommaOk {
				return fmt.Sprintf("((void (%s)), [undefined, false] as readonly [%s, boolean])", x, targetUnion), true, nil
			}
			return fmt.Sprintf("(($i: %s): %s => { goif$.goPanicConversionIface($i, %q, %q, %s); })(%s)",
				union, targetUnion, n.SourceDisplay, n.TargetDisplay, requiredList(n.Required), x), true, nil
		}
		if n.CommaOk {
			return fmt.Sprintf("(($i: %s): readonly [%s, boolean] => (%s ? [$i, true] : [undefined, false]))(%s)",
				union, targetUnion, condition, x), true, nil
		}
		return fmt.Sprintf("(($i: %s): %s => { if (%s) { return $i; } goif$.goPanicConversionIface($i, %q, %q, %s); })(%s)",
			union, targetUnion, condition, n.SourceDisplay, n.TargetDisplay, requiredList(n.Required), x), true, nil
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
	// iface.M(g()): the sole argument is a multi-result spread — bind the
	// tuple once (after the receiver, before the nil panic) and spread it
	// into every narrowed dispatch, so f(g()) lowers on the interface call
	// path too.
	spreadTuple, isSpread, err := p.spreadInner(n.Args)
	if err != nil {
		return "", err
	}
	var argNames []string
	if isSpread {
		argNames = []string{"...$t"}
	} else {
		argNames = make([]string, len(n.Args))
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
	}
	result, err := p.tsFuncResultType(n.Results)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	sub := &printer{out: &body, module: p.module, indent: 0}
	if isSpread {
		sub.line("const $t = %s;", spreadTuple)
	}
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
		// Dispatch indexes the member's CANONICAL SLOT for this interface
		// method — IfaceMember.Slots is keyed by the interface method's
		// canonical identity (ir.MethodKey) and holds the member's
		// ir.MethodSlot for its implementing method. Membership is gated by
		// types.Implements, so every retained member populates a slot for
		// this method; a missing slot is a construction-invariant violation,
		// NOT a bare-name fallback.
		selector, ok := member.Slots[n.MethodKey]
		if !ok {
			panic("emit: interface member " + member.K + " has no dispatch slot for the called method")
		}
		call := "$r.m." + selector + "(" + joinComma(append([]string{"$r.v"}, argNames...)) + ")"
		if result == "void" {
			sub.line("case %q: %s; return;", member.K, call)
		} else {
			sub.line("case %q: return %s;", member.K, call)
		}
	}
	sub.line("default: gort$.goPanicUnreachableType(%q);", recvType.Go)
	sub.indent--
	sub.line("}")
	closing := strings.Repeat("  ", p.indent)
	return fmt.Sprintf("((%s): %s => {\n%s%s})(%s)",
		joinComma(params), result, body.String(), closing, joinComma(passed)), nil
}

// requiredList spells the target interface's method displays.
func requiredList(required []string) string {
	quoted := make([]string, len(required))
	for i, name := range required {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	return "[" + joinComma(quoted) + "]"
}

// assertTokenPossible reports whether a discriminant token is a member of
// the operand union: a token outside the closed member set (a type never
// boxed in the unit, or from a non-materialized package) can never exist
// at runtime, so the assertion is statically false and its comparison
// must not be emitted.
func (p *printer) assertTokenPossible(unionT ir.Type, token string) bool {
	for _, member := range p.retainedMembers(unionT) {
		if member.K == token {
			return true
		}
	}
	if unionT.IfaceEmpty {
		for _, member := range predeclaredMembers {
			if "p:"+member.name == token {
				return true
			}
		}
		for _, composite := range p.module.BoxedComposites {
			if p.referencesWithheldType(composite.T) {
				continue
			}
			if "c:"+composite.Canon == token {
				return true
			}
		}
	}
	return false
}
