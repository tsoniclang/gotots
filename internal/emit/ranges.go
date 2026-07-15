// Range-statement emission: slices, strings (rune decoding), maps
// (deterministic insertion order — one of Go's permitted orders), and
// integer ranges, each with per-iteration variable copies.
package emit

import (
	"github.com/tsoniclang/gotots/internal/ir"
)

// printRangeSlice emits range-over-slice with the operand and length
// evaluated once and per-iteration element loads.
func (p *printer) printRangeSlice(n *ir.RangeSlice) error {
	operand, err := p.printExpr(n.X)
	if err != nil {
		return err
	}
	sliceTemp := p.temp()
	p.line("const %s = %s;", sliceTemp, operand)
	lengthTemp := p.temp()
	p.line("const %s = gosl$.goSliceLen(%s);", lengthTemp, sliceTemp)
	// The induction variable is always hidden: the source index name
	// binds a per-iteration copy, so assigning to it inside the body
	// never affects iteration — exactly Go.
	induction := p.temp()
	p.line("for (let %s: goabi$.GoInt = 0n; %s < %s; %s = %s + 1n) {", induction, induction, lengthTemp, induction, induction)
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
			p.line("let %s: %s = gosl$.goSliceGet(%s, %s).goClone$();", tsName(n.Value), spelled, sliceTemp, induction)
		} else if n.VarT.Kind == ir.KindArray {
			cloneElem, err := p.arrayElemClone(*n.VarT.Elem)
			if err != nil {
				return err
			}
			p.line("let %s: %s = gosl$.goArrayClone(gosl$.goSliceGet(%s, %s), %s);", tsName(n.Value), spelled, sliceTemp, induction, cloneElem)
		} else {
			p.line("let %s: %s = gosl$.goSliceGet(%s, %s);", tsName(n.Value), spelled, sliceTemp, induction)
		}
	}
	if err := p.printBlockBody(n.Body); err != nil {
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
	p.line("for (const %s of gort$.goStringRange(%s)) {", entry, operand)
	p.indent++
	if n.Index != "" {
		p.line("let %s: goabi$.GoInt = %s[0];", tsName(n.Index), entry)
	}
	if n.Value != "" {
		p.line("let %s: number = %s[1];", tsName(n.Value), entry)
	}
	if err := p.printBlockBody(n.Body); err != nil {
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
	mapTemp := p.temp()
	p.line("const %s = %s;", mapTemp, operand)
	entry := p.temp()
	p.line("for (const %s of (%s === undefined ? [] : %s)) {", entry, mapTemp, mapTemp)
	p.indent++
	if n.Key != "" {
		p.line("let %s = %s[0];", tsName(n.Key), entry)
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
		}
		p.line("let %s: %s = %s;", tsName(n.Value), spelled, value)
	}
	if err := p.printBlockBody(n.Body); err != nil {
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
		p.line("for (let %s: %s = 0n; %s < %s; %s = %s + 1n) {", induction, spelled, induction, limit, induction, induction)
	} else {
		p.line("for (let %s: %s = 0; %s < %s; %s = %s + 1) {", induction, spelled, induction, limit, induction, induction)
	}
	p.indent++
	if n.Index != "" {
		p.line("let %s: %s = %s;", tsName(n.Index), spelled, induction)
	}
	if err := p.printBlockBody(n.Body); err != nil {
		return err
	}
	p.indent--
	p.line("}")
	return nil
}
