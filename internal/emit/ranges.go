// Range-statement emission: slices, strings (rune decoding), maps
// (deterministic insertion order — one of Go's permitted orders), and
// integer ranges, each with per-iteration variable copies.
package emit

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printRangeSlice emits range-over-slice with the operand and length
// evaluated once and per-iteration element loads.
func (p *printer) printRangeSlice(n *ir.RangeSlice) error {
	operand, err := p.printExpr(n.X)
	if err != nil {
		return err
	}
	get, length := "gosl$.goSliceGet", "gosl$.goSliceLen"
	if p.nativeSlice(n.X) {
		get, length = "gosl$.goArrayGet", "gosl$.goArrayLen"
	}
	sliceTemp := p.temp()
	p.line("const %s = %s;", sliceTemp, operand)
	lengthTemp := p.temp()
	p.line("const %s = %s(%s);", lengthTemp, length, sliceTemp)
	// The induction variable is always hidden: the source index name
	// binds a per-iteration copy, so assigning to it inside the body
	// never affects iteration — exactly Go.
	induction := p.temp()
	p.line("%sfor (let %s: goabi$.GoInt = 0n; %s < %s; %s = %s + 1n) {", p.takeLoopLabel(), induction, induction, lengthTemp, induction, induction)
	p.indent++
	if n.Index != "" {
		p.line("let %s: goabi$.GoInt = %s;", tsName(n.Index), induction)
	}
	if n.Value != "" {
		spelled, err := p.tsType(n.VarT)
		if err != nil {
			return err
		}
		if n.VarT.Kind == ir.KindStruct {
			// The range variable binds a per-iteration value copy.
			p.line("let %s: %s = %s(%s, %s).goClone$();", tsName(n.Value), spelled, get, sliceTemp, induction)
		} else if n.VarT.Kind == ir.KindArray {
			cloneElem, err := p.arrayElemClone(*n.VarT.Elem)
			if err != nil {
				return err
			}
			p.line("let %s: %s = gosl$.goArrayClone(%s(%s, %s), %s);", tsName(n.Value), spelled, get, sliceTemp, induction, cloneElem)
		} else if n.VarT.Kind == ir.KindExternal {
			callee, err := p.module.symbol(n.VarT.Pkg, externCloneSymbol(n.VarT.Named))
			if err != nil {
				return err
			}
			p.line("let %s: %s = %s(%s(%s, %s));", tsName(n.Value), spelled, callee, get, sliceTemp, induction)
		} else {
			p.line("let %s: %s = %s(%s, %s);", tsName(n.Value), spelled, get, sliceTemp, induction)
		}
	}
	if err := p.printLoopBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// printRangeString emits range over a string: the operand evaluates
// once; each iteration binds the starting byte offset and the decoded
// rune. Strings are immutable, so the decoded sequence is a snapshot by
// construction.
func (p *printer) printRangeString(n *ir.RangeString) error {
	operand, err := p.printExpr(n.X)
	if err != nil {
		return err
	}
	entry := p.temp()
	p.line("%sfor (const %s of gort$.goStringRange(%s)) {", p.takeLoopLabel(), entry, operand)
	p.indent++
	if n.Index != "" {
		p.line("let %s: goabi$.GoInt = %s[0];", tsName(n.Index), entry)
	}
	if n.Value != "" {
		p.line("let %s: number = %s[1];", tsName(n.Value), entry)
	}
	if err := p.printLoopBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// printRangeMap emits range over a map: live entries in the carrier's
// deterministic insertion order — one of Go's permitted orders. Struct
// and array values bind per-iteration copies.
func (p *printer) printRangeMap(n *ir.RangeMap) error {
	operand, err := p.printExpr(n.X)
	if err != nil {
		return err
	}
	keyed := n.X.Type().Key.Kind == ir.KindStruct
	float := n.X.Type().Key.Kind.Float()
	encoded := n.X.Type().Key.Kind == ir.KindIface && (n.X.Type().Key.TypeParamName == "" || p.familyEnc)
	mapTemp := p.temp()
	// The temp declares its full nilable map type so a provably nil
	// operand (legal Go: range over a nil map iterates zero times) does
	// not narrow to literal undefined.
	spelledMap, err := p.tsType(n.X.Type())
	if err != nil {
		return err
	}
	p.line("const %s: %s = %s;", mapTemp, spelledMap, operand)
	entry := p.temp()
	if keyed {
		p.line("%sfor (const %s of gort$.goKMapValues(%s)) {", p.takeLoopLabel(), entry, mapTemp)
	} else if encoded {
		p.line("%sfor (const %s of gort$.goEMapValues(%s)) {", p.takeLoopLabel(), entry, mapTemp)
	} else if float {
		p.line("%sfor (const %s of gort$.goFMapEntries(%s)) {", p.takeLoopLabel(), entry, mapTemp)
	} else {
		p.line("%sfor (const %s of gort$.goMapEntries(%s)) {", p.takeLoopLabel(), entry, mapTemp)
	}
	p.indent++
	if n.Key != "" {
		if keyed {
			// The stored struct key binds as a per-iteration value copy.
			p.line("let %s = %s[0].goClone$();", tsName(n.Key), entry)
		} else {
			p.line("let %s = %s[0];", tsName(n.Key), entry)
		}
	}
	if n.Value != "" {
		spelled, err := p.tsType(n.ValT)
		if err != nil {
			return err
		}
		value := entry + "[1]"
		switch n.ValT.Kind {
		case ir.KindStruct:
			value += ".goClone$()"
		case ir.KindArray:
			cloneElem, err := p.arrayElemClone(*n.ValT.Elem)
			if err != nil {
				return err
			}
			value = "gosl$.goArrayClone(" + value + ", " + cloneElem + ")"
		case ir.KindExternal:
			callee, err := p.module.symbol(n.ValT.Pkg, externCloneSymbol(n.ValT.Named))
			if err != nil {
				return err
			}
			value = fmt.Sprintf("%s(%s)", callee, value)
		}
		p.line("let %s: %s = %s;", tsName(n.Value), spelled, value)
	}
	if err := p.printLoopBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// printRangeInt emits `for i := range n`: n evaluates once and the
// index counts in n's own carrier.
func (p *printer) printRangeInt(n *ir.RangeInt) error {
	operand, err := p.printExpr(n.N)
	if err != nil {
		return err
	}
	limit := p.temp()
	p.line("const %s = %s;", limit, operand)
	spelled, err := p.tsType(n.N.Type())
	if err != nil {
		return err
	}
	// The induction variable is always hidden (the source name binds a
	// per-iteration copy); bounds keep increments exact without wrap
	// helpers, since the index stays strictly below the operand.
	induction := p.temp()
	if n.N.Type().Kind.Wide64() {
		p.line("%sfor (let %s: %s = 0n; %s < %s; %s = %s + 1n) {", p.takeLoopLabel(), induction, spelled, induction, limit, induction, induction)
	} else {
		p.line("%sfor (let %s: %s = 0; %s < %s; %s = %s + 1) {", p.takeLoopLabel(), induction, spelled, induction, limit, induction, induction)
	}
	p.indent++
	if n.Index != "" {
		p.line("let %s: %s = %s;", tsName(n.Index), spelled, induction)
	}
	if err := p.printLoopBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}

// printRangeFunc emits range over a function iterator: the sequence
// runs once with a synthesized yield closure. Break and continue in the
// body follow the yield protocol, a body return is captured and
// re-issued after the sequence unwinds, and a yield after loop exit
// panics with Go's exact runtime message.
func (p *printer) printRangeFunc(n *ir.RangeFunc) error {
	seq, err := p.printExpr(n.Seq)
	if err != nil {
		return err
	}
	ctx := &rangeFuncCtx{doneVar: p.temp(), returnedVar: p.temp()}
	p.line("let %s = false;", ctx.doneVar)
	p.line("let %s = false;", ctx.returnedVar)
	resultSpelled := ""
	if len(n.Results) > 0 {
		if resultSpelled, err = p.tsFuncResultType(n.Results); err != nil {
			return err
		}
		// A one-field slot rather than a bare let: property reads are not
		// control-flow narrowed, so the propagated value keeps its type
		// even when TypeScript cannot see the closure assignment.
		ctx.retVar = p.temp()
		p.line("const %s: { v?: (%s) } = {};", ctx.retVar, resultSpelled)
	}
	params := make([]string, 0, 2)
	yieldArgs := []struct {
		name string
		t    ir.Type
	}{{n.Key, n.KeyT}}
	if n.TwoVars {
		yieldArgs = append(yieldArgs, struct {
			name string
			t    ir.Type
		}{n.Value, n.ValT})
	}
	for i, arg := range yieldArgs {
		spelled, err := p.tsType(arg.t)
		if err != nil {
			return err
		}
		params = append(params, fmt.Sprintf("$y%d: %s", i, spelled))
	}
	// A nil sequence function panics exactly like Go's nil call.
	checkedSeq, err := p.nilCheckOf(seq, n.Seq.Type())
	if err != nil {
		return err
	}
	p.line("(%s)((%s): boolean => {", checkedSeq, joinComma(params))
	p.indent++
	p.line("if (%s) {", ctx.doneVar)
	p.indent++
	p.line("gort$.goPanicRangeExit();")
	p.indent--
	p.line("}")
	for i, arg := range yieldArgs {
		if arg.name == "" {
			continue
		}
		value := fmt.Sprintf("$y%d", i)
		switch arg.t.Kind {
		case ir.KindStruct:
			value += ".goClone$()"
		case ir.KindArray:
			cloneElem, err := p.arrayElemClone(*arg.t.Elem)
			if err != nil {
				return err
			}
			value = "gosl$.goArrayClone(" + value + ", " + cloneElem + ")"
		}
		p.line("let %s = %s;", tsName(arg.name), value)
	}
	savedBreak, savedContinue, savedReturn := p.rangeBreak, p.rangeContinue, p.rangeReturn
	p.rangeBreak, p.rangeContinue, p.rangeReturn = ctx, ctx, ctx
	err = p.printBlockBody(n.Body)
	p.rangeBreak, p.rangeContinue, p.rangeReturn = savedBreak, savedContinue, savedReturn
	if err != nil {
		return err
	}
	p.line("return true;")
	p.indent--
	p.line("});")
	// The sequence has returned: any subsequent call of a retained yield
	// is invalid, exactly as Go marks the iterator done.
	p.line("%s = true;", ctx.doneVar)
	// The propagated-return epilogue exists only when some body return
	// actually set it: with no body return the flag and slot are never
	// assigned, and TypeScript correctly narrows them to their initial
	// undefined/false.
	if ctx.returnUsed {
		p.line("if (%s) {", ctx.returnedVar)
		p.indent++
		if ctx.retVar != "" {
			p.emitFunctionReturn("(" + ctx.retVar + ".v as (" + resultSpelled + "))")
		} else {
			p.emitFunctionReturn("")
		}
		p.indent--
		p.line("}")
	}
	return nil
}
