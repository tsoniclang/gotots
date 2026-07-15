// Struct-class emission: the class with its total constructor, the
// value-semantics contract (clone / in-place set / fresh zero / the
// canonical key encoding when every field is encodable), and method
// functions with explicit receivers.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
)

// printStruct emits one named struct as a class whose constructor takes
// every field in declaration order (composite literals pass explicit
// zeros for omitted fields, so construction is always total).
func printStruct(out *strings.Builder, module *Module, structDecl *ir.Struct) error {
	p := &printer{out: out, module: module}
	export := ""
	if structDecl.Exported {
		export = "export "
	}
	p.line("%sclass %s {", export, tsName(structDecl.Name))
	p.indent++
	var params []string
	for _, field := range structDecl.Fields {
		spelled, err := p.tsType(field.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", structDecl.ID, err)
		}
		p.line("%s: %s;", field.Name, spelled)
		params = append(params, tsName(field.Name)+": "+spelled)
	}
	p.line("constructor(%s) {", strings.Join(params, ", "))
	p.indent++
	for _, field := range structDecl.Fields {
		p.line("this.%s = %s;", field.Name, tsName(field.Name))
	}
	p.indent--
	p.line("}")
	if err := printStructValueContract(p, structDecl); err != nil {
		return fmt.Errorf("%s: %w", structDecl.ID, err)
	}
	p.indent--
	p.line("}")
	return nil
}

// printStructValueContract emits the value-semantics members of every
// struct class: a deep clone along the value-struct spine, an in-place
// field overwrite (so aliases of the value observe whole-value stores,
// exactly like Go's memory write), and a fresh zero value. The names
// carry a "$" that no Go identifier can spell, so no source method can
// collide.
func printStructValueContract(p *printer, structDecl *ir.Struct) error {
	clone := make([]string, 0, len(structDecl.Fields))
	for _, field := range structDecl.Fields {
		switch field.Type.Kind {
		case ir.KindStruct:
			clone = append(clone, "this."+field.Name+".goClone$()")
		case ir.KindArray:
			cloneElem, err := p.arrayElemClone(*field.Type.Elem)
			if err != nil {
				return err
			}
			clone = append(clone, "gosl$.goArrayClone(this."+field.Name+", "+cloneElem+")")
		case ir.KindExternal:
			spelled, err := p.tsType(field.Type)
			if err != nil {
				return err
			}
			clone = append(clone, fmt.Sprintf("(goext$.goExternalCall(%q, [this.%s]) as %s)",
				field.Type.Pkg+"."+field.Type.Named+".goClone$", field.Name, spelled))
		default:
			clone = append(clone, "this."+field.Name)
		}
	}
	p.line("goClone$(): %s {", tsName(structDecl.Name))
	p.indent++
	p.line("return new %s(%s);", tsName(structDecl.Name), strings.Join(clone, ", "))
	p.indent--
	p.line("}")

	p.line("goSet$(other: %s): void {", tsName(structDecl.Name))
	p.indent++
	for _, field := range structDecl.Fields {
		switch field.Type.Kind {
		case ir.KindStruct:
			p.line("this.%s.goSet$(other.%s);", field.Name, field.Name)
		case ir.KindArray:
			setElem, err := p.arrayElemSet(*field.Type.Elem)
			if err != nil {
				return err
			}
			p.line("gosl$.goArraySetAll(this.%s, other.%s, %s);", field.Name, field.Name, setElem)
		case ir.KindExternal:
			p.line("goext$.goExternalCall(%q, [this.%s, other.%s]);",
				field.Type.Pkg+"."+field.Type.Named+".goSet$", field.Name, field.Name)
		default:
			p.line("this.%s = other.%s;", field.Name, field.Name)
		}
	}
	p.indent--
	p.line("}")

	zeros := make([]string, 0, len(structDecl.Fields))
	for _, field := range structDecl.Fields {
		zero, err := p.zeroLiteral(field.Type)
		if err != nil {
			return err
		}
		zeros = append(zeros, zero)
	}
	p.line("static goZero$(): %s {", tsName(structDecl.Name))
	p.indent++
	p.line("return new %s(%s);", tsName(structDecl.Name), strings.Join(zeros, ", "))
	p.indent--
	p.line("}")

	// Structs whose comparable fields all encode deterministically carry
	// goKey$ — the injective canonical key for the keyed-map carrier.
	encodable := true
	for _, field := range structDecl.Fields {
		if !ir.KeyEncodableField(field.Type) {
			encodable = false
			break
		}
	}
	if encodable {
		components := make([]string, 0, len(structDecl.Fields))
		for _, field := range structDecl.Fields {
			component, err := keyComponent("this."+field.Name, field.Type)
			if err != nil {
				return err
			}
			components = append(components, component)
		}
		if len(components) == 0 {
			components = append(components, `"z"`)
		}
		p.line("goKey$(): string {")
		p.indent++
		p.line("return %s;", strings.Join(components, ` + "|" + `))
		p.indent--
		p.line("}")
	}
	return nil
}

// keyComponent spells one field's deterministic key encoding: each
// component parses left to right unambiguously, so the composition is
// injective (strings are length-prefixed; array lengths are fixed by
// the type).
func keyComponent(access string, t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindString:
		return `String(` + access + `.length) + ":" + ` + access, nil
	case t.Kind == ir.KindBool:
		return `(` + access + ` ? "t" : "f")`, nil
	case t.Kind == ir.KindPointer:
		return "gort$.goKeyId(" + access + ")", nil
	case t.Kind == ir.KindUnit:
		return `"z"`, nil
	case t.Kind == ir.KindArray:
		elem, err := keyComponent("$v", *t.Elem)
		if err != nil {
			return "", err
		}
		return "gort$.goKeyArray(" + access + ", ($v) => " + elem + ")", nil
	case t.Kind.Integer():
		return `"i" + String(` + access + `)`, nil
	}
	return "", fmt.Errorf("no key encoding for field type %q", t.Go)
}

// printMethodFunction emits one statically resolved method as a
// package-level function named Type$Method ("$" spells outside Go's
// identifier alphabet, so the name can never collide). A pointer
// receiver is the first parameter, nilable — Go runs pointer-receiver
// methods on nil receivers and only the body's dereferences panic. A
// struct value receiver clones on entry, so the caller's instance never
// mutates; scalar value receivers copy at the call like every JS value.
func printMethodFunction(out *strings.Builder, module *Module, className string, method *ir.Func) error {
	p := &printer{out: out, module: module}
	recvSpelled, err := p.tsType(method.Receiver.Type)
	if err != nil {
		return fmt.Errorf("%s: %w", method.ID, err)
	}
	structValueReceiver := method.Receiver.Type.Kind == ir.KindStruct
	arrayValueReceiver := method.Receiver.Type.Kind == ir.KindArray
	recvParam := tsName(method.Receiver.Name)
	if structValueReceiver || arrayValueReceiver {
		recvParam = "$recv"
	}
	params := []string{recvParam + ": " + recvSpelled}
	for _, parameter := range method.Params {
		spelled, err := p.tsType(parameter.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", method.ID, err)
		}
		params = append(params, tsName(parameter.Name)+": "+spelled)
	}
	result, err := p.tsResultType(method.Results)
	if err != nil {
		return fmt.Errorf("%s: %w", method.ID, err)
	}
	export := ""
	if method.Exported {
		export = "export "
	}
	p.line("%sfunction %s$%s(%s): %s {", export, className, method.Name, strings.Join(params, ", "), result)
	p.indent++
	if structValueReceiver {
		p.line("const %s = $recv.goClone$();", tsName(method.Receiver.Name))
	}
	if arrayValueReceiver {
		cloneElem, err := p.arrayElemClone(*method.Receiver.Type.Elem)
		if err != nil {
			return fmt.Errorf("%s: %w", method.ID, err)
		}
		p.line("const %s = gosl$.goArrayClone($recv, %s);", tsName(method.Receiver.Name), cloneElem)
	}
	if err := p.printDeferWrappedBody(method.Body, method.UsesDeferStack); err != nil {
		return fmt.Errorf("%s: %w", method.ID, err)
	}
	p.indent--
	p.line("}")
	return nil
}

// printDeferWrappedBody prints a function body, wrapping it — when the
// body uses the defer stack — in one try/finally that drains deferred
// calls in LIFO order at every exit.
func (p *printer) printDeferWrappedBody(body *ir.Block, usesDeferStack bool) error {
	if !usesDeferStack {
		return p.printBlockBody(body)
	}
	p.line("const _ds$: (() => void)[] = [];")
	p.line("try {")
	p.indent++
	if err := p.printBlockBody(body); err != nil {
		return err
	}
	p.indent--
	p.line("} finally {")
	p.indent++
	p.line("for (let _di$ = _ds$.length - 1; _di$ >= 0; _di$--) {")
	p.indent++
	p.line("(_ds$[_di$] as () => void)();")
	p.indent--
	p.line("}")
	p.indent--
	p.line("}")
	return nil
}
