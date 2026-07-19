// Struct-class emission: the class with its total constructor, the
// value-semantics contract (clone / in-place set / fresh zero / the
// canonical key encoding when every field is encodable), and method
// functions with explicit receivers.
package emit

import (
	"fmt"
	"strings"

	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/tsident"
)

// printStruct emits one named struct as a class whose constructor takes
// every field in declaration order (composite literals pass explicit
// zeros for omitted fields, so construction is always total).
func printStruct(out *strings.Builder, module *Module, structDecl *ir.Struct) error {
	p := &printer{out: out, module: module}
	export := "export "
	generics := ""
	if len(structDecl.TypeParams) > 0 {
		generics = "<" + strings.Join(structDecl.TypeParams, ", ") + ">"
	}
	p.line("%sclass %s%s {", export, tsName(structDecl.Name), generics)
	p.indent++
	var params []string
	for _, field := range structDecl.Fields {
		spelled, err := p.tsType(field.Type)
		if err != nil {
			return fmt.Errorf("%s: %w", structDecl.ID, err)
		}
		// An address-taken field is one stable per-instance cell; the
		// declared field is GoCell<T> but the constructor still takes the
		// plain value and wraps it, so every construction site is unchanged.
		if field.Cell {
			p.line("%s: gort$.GoCell<%s>;", field.Name, spelled)
		} else {
			p.line("%s: %s;", field.Name, spelled)
		}
		params = append(params, tsName(field.Name)+": "+spelled)
	}
	// A generic class captures its instantiation's copy factories at
	// construction: clone$P/set$P fields keep goClone$/goSet$ zero-arg (the
	// structural GoStructValue contract), while their behavior stays exact
	// per instantiation.
	for i, param := range structDecl.TypeParams {
		p.line("eq$%s: (a: %s, b: %s) => boolean;", param, param, param)
		p.line("clone$%s: (v: %s) => %s;", param, param, param)
		p.line("set$%s: ((d: %s, s: %s) => void) | undefined;", param, param, param)
		params = append(params, "eq$"+param+": (a: "+param+", b: "+param+") => boolean")
		params = append(params, "clone$"+param+": (v: "+param+") => "+param)
		params = append(params, "set$"+param+": ((d: "+param+", s: "+param+") => void) | undefined")
		if structKeyed(structDecl, i) {
			p.line("key$%s: (k: %s) => string;", param, param)
			params = append(params, "key$"+param+": (k: "+param+") => string")
		}
	}
	p.line("constructor(%s) {", strings.Join(params, ", "))
	p.indent++
	for _, field := range structDecl.Fields {
		if field.Cell {
			p.line("this.%s = { v: %s };", field.Name, tsName(field.Name))
		} else {
			p.line("this.%s = %s;", field.Name, tsName(field.Name))
		}
	}
	for i, param := range structDecl.TypeParams {
		p.line("this.eq$%s = eq$%s;", param, param)
		p.line("this.clone$%s = clone$%s;", param, param)
		p.line("this.set$%s = set$%s;", param, param)
		if structKeyed(structDecl, i) {
			p.line("this.key$%s = key$%s;", param, param)
		}
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
		// A cell field is a non-identity carrier: the clone passes its held
		// value, and the constructor wraps it in a FRESH cell — so the
		// clone's field has its own distinct address, exactly Go's copy.
		if field.Cell {
			clone = append(clone, "this."+field.Name+".v")
			continue
		}
		if field.Type.Kind == ir.KindIface && field.Type.TypeParamName != "" {
			clone = append(clone, "this.clone$"+field.Type.TypeParamName+"(this."+field.Name+")")
			continue
		}
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
			callee, err := p.module.symbol(field.Type.Pkg, externCloneSymbol(field.Type.Named))
			if err != nil {
				return err
			}
			clone = append(clone, fmt.Sprintf("%s(this.%s)", callee, field.Name))
		default:
			clone = append(clone, "this."+field.Name)
		}
	}
	self := tsName(structDecl.Name)
	if len(structDecl.TypeParams) > 0 {
		self += "<" + strings.Join(structDecl.TypeParams, ", ") + ">"
	}
	for i, param := range structDecl.TypeParams {
		clone = append(clone, "this.eq$"+param, "this.clone$"+param, "this.set$"+param)
		if structKeyed(structDecl, i) {
			clone = append(clone, "this.key$"+param)
		}
	}
	p.line("goClone$(): %s {", self)
	p.indent++
	p.line("return new %s(%s);", self, strings.Join(clone, ", "))
	p.indent--
	p.line("}")

	p.line("goSet$(other: %s): void {", self)
	p.indent++
	for _, field := range structDecl.Fields {
		// A cell field keeps its stable storage under a whole-value store:
		// only the held value is overwritten, so &this.f stays exact.
		if field.Cell {
			p.line("this.%s.v = other.%s.v;", field.Name, field.Name)
			continue
		}
		if field.Type.Kind == ir.KindIface && field.Type.TypeParamName != "" {
			param := field.Type.TypeParamName
			p.line("if (this.set$%s === undefined) { this.%s = other.%s; } else { this.set$%s(this.%s, other.%s); }",
				param, field.Name, field.Name, param, field.Name, field.Name)
			continue
		}
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
			callee, err := p.module.symbol(field.Type.Pkg, externSetSymbol(field.Type.Named))
			if err != nil {
				return err
			}
			p.line("%s(this.%s, other.%s);", callee, field.Name, field.Name)
		default:
			p.line("this.%s = other.%s;", field.Name, field.Name)
		}
	}
	p.indent--
	p.line("}")

	// A generic class's zero needs one factory per type parameter: a
	// bare-parameter field's zero depends on the instantiation.
	savedFactories, savedEqOps, savedClone, savedSet, savedKey := p.zeroFactories, p.eqOps, p.cloneOps, p.setOps, p.keyOps
	if len(structDecl.TypeParams) > 0 {
		p.zeroFactories = map[string]string{}
		p.eqOps = map[string]string{}
		p.cloneOps = map[string]string{}
		p.setOps = map[string]string{}
		p.keyOps = map[string]string{}
		for i, param := range structDecl.TypeParams {
			p.zeroFactories[param] = "zero$" + param
			p.eqOps[param] = "eq$" + param
			p.cloneOps[param] = "clone$" + param
			p.setOps[param] = "set$" + param
			if structKeyed(structDecl, i) {
				p.keyOps[param] = "key$" + param
			}
		}
	}
	zeros := make([]string, 0, len(structDecl.Fields))
	for _, field := range structDecl.Fields {
		zero, err := p.zeroLiteral(field.Type)
		if err != nil {
			p.zeroFactories, p.eqOps, p.cloneOps, p.setOps, p.keyOps = savedFactories, savedEqOps, savedClone, savedSet, savedKey
			return err
		}
		zeros = append(zeros, zero)
	}
	p.zeroFactories, p.eqOps, p.cloneOps, p.setOps, p.keyOps = savedFactories, savedEqOps, savedClone, savedSet, savedKey
	if len(structDecl.TypeParams) > 0 {
		factories := make([]string, 0, len(structDecl.TypeParams)*4)
		for _, param := range structDecl.TypeParams {
			factories = append(factories, "zero$"+param+": () => "+param)
		}
		for i, param := range structDecl.TypeParams {
			factories = append(factories, "eq$"+param+": (a: "+param+", b: "+param+") => boolean")
			factories = append(factories, "clone$"+param+": (v: "+param+") => "+param)
			factories = append(factories, "set$"+param+": ((d: "+param+", s: "+param+") => void) | undefined")
			if structKeyed(structDecl, i) {
				factories = append(factories, "key$"+param+": (k: "+param+") => string")
			}
		}
		ctorArgs := append([]string{}, zeros...)
		for i, param := range structDecl.TypeParams {
			ctorArgs = append(ctorArgs, "eq$"+param, "clone$"+param, "set$"+param)
			if structKeyed(structDecl, i) {
				ctorArgs = append(ctorArgs, "key$"+param)
			}
		}
		p.line("static goZero$<%s>(%s): %s {", strings.Join(structDecl.TypeParams, ", "), strings.Join(factories, ", "), self)
		p.indent++
		p.line("return new %s(%s);", self, strings.Join(ctorArgs, ", "))
		p.indent--
		p.line("}")
	} else {
		p.line("static goZero$(): %s {", self)
		p.indent++
		p.line("return new %s(%s);", self, strings.Join(zeros, ", "))
		p.indent--
		p.line("}")
	}

	// Comparable structs carry goEq$: exact field-wise equality (floats
	// keep NaN and signed-zero semantics under ===; interface fields may
	// panic, exactly Go).
	if structDecl.Comparable {
		savedEq := p.eqOps
		if len(structDecl.TypeParams) > 0 {
			p.eqOps = map[string]string{}
			for _, param := range structDecl.TypeParams {
				p.eqOps[param] = "this.eq$" + param
			}
		}
		comparisons := make([]string, 0, len(structDecl.Fields))
		for _, field := range structDecl.Fields {
			this, other := "this."+field.Name, "other."+field.Name
			if field.Cell {
				this += ".v"
				other += ".v"
			}
			comparison, err := p.eqComponent(this, other, field.Type)
			if err != nil {
				p.eqOps = savedEq
				return err
			}
			comparisons = append(comparisons, comparison)
		}
		if len(comparisons) == 0 {
			comparisons = append(comparisons, "true")
		}
		p.eqOps = savedEq
		p.line("goEq$(other: %s): boolean {", self)
		p.indent++
		p.line("return %s;", strings.Join(comparisons, " && "))
		p.indent--
		p.line("}")
	}

	// Structs whose comparable fields all encode deterministically carry
	// goKey$ — the injective canonical key for the keyed-map carrier.
	// KeyEncodable is the IR verdict (it descends into nested struct and
	// array-of-struct fields, which the pure field predicate cannot).
	if structDecl.KeyEncodable {
		components := make([]string, 0, len(structDecl.Fields))
		for _, field := range structDecl.Fields {
			access := "this." + field.Name
			if field.Cell {
				access += ".v"
			}
			component, err := p.keyComponent(access, field.Type)
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
func (p *printer) keyComponent(access string, t ir.Type) (string, error) {
	switch {
	case t.Kind == ir.KindString:
		return tsident.Global("String") + "(" + access + `.length) + ":" + ` + access, nil
	case t.Kind == ir.KindBool:
		return `(` + access + ` ? "t" : "f")`, nil
	case t.Kind == ir.KindPointer:
		return "gort$.goKeyId(" + access + ")", nil
	case t.Kind == ir.KindUnit:
		return `"z"`, nil
	case t.Kind == ir.KindArray:
		elem, err := p.keyComponent("$v", *t.Elem)
		if err != nil {
			return "", err
		}
		return "gort$.goKeyArray(" + access + ", ($v) => " + elem + ")", nil
	case t.Kind == ir.KindStruct:
		// A nested key struct composes through its own goKey$ (guaranteed
		// present: the IR admits the outer key only when every nested
		// struct field is itself key-encodable). Wrapped so the nested
		// encoding parses unambiguously inside the outer composition.
		return `"{" + ` + access + `.goKey$() + "}"`, nil
	case t.Kind.Integer():
		return `"i" + ` + tsident.Global("String") + "(" + access + ")", nil
	case t.Kind.Float():
		return "gort$.goKeyFloat(" + access + ")", nil
	case t.Kind == ir.KindIface && t.TypeParamName != "":
		// A bare type-parameter field encodes through the captured key
		// operation — statically exact per instantiation.
		return "this.key$" + t.TypeParamName + "(" + access + ")", nil
	case t.Kind == ir.KindIface && t.TypeParamName == "":
		// The union's generated $key encoder carries Go's dynamic-key
		// equality for the interface field.
		name, err := p.ifaceUnionAlias(t)
		if err != nil {
			return "", err
		}
		p.module.RequireIfaceKeyFn(name)
		return name + "$key(" + access + ")", nil
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
	// The instantiation's zero and equality operations, mirroring
	// generic function signatures.
	for _, param := range method.TypeParams {
		params = append(params, "zero$"+param+": () => "+param)
	}
	for _, param := range method.TypeParams {
		params = append(params, "eq$"+param+": (a: "+param+", b: "+param+") => boolean")
	}
	for _, param := range method.TypeParams {
		params = append(params, "clone$"+param+": (v: "+param+") => "+param)
	}
	for _, param := range method.TypeParams {
		params = append(params, "set$"+param+": ((d: "+param+", s: "+param+") => void) | undefined")
	}
	for i, param := range method.TypeParams {
		if i < len(method.KeyedParams) && method.KeyedParams[i] {
			params = append(params, "key$"+param+": (k: "+param+") => string")
		}
	}
	result, err := p.tsResultType(method.Results)
	if err != nil {
		return fmt.Errorf("%s: %w", method.ID, err)
	}
	export := "export "
	generics := ""
	if len(method.TypeParams) > 0 {
		generics = "<" + strings.Join(method.TypeParams, ", ") + ">"
	}
	p.slicePlans = method.SlicePlans
	p.line("%sfunction %s$%s%s(%s): %s {", export, className, method.Name, generics, strings.Join(params, ", "), result)
	p.indent++
	if method.Placeholder {
		// A materialized placeholder renders its exact signature and a
		// fail-closed throw only — never the receiver preamble or the
		// (absent or partial) real body.
		p.printPlaceholderBody(method.ID)
		p.indent--
		p.line("}")
		return nil
	}
	if len(method.TypeParams) > 0 {
		p.zeroFactories = map[string]string{}
		p.eqOps = map[string]string{}
		p.cloneOps = map[string]string{}
		p.setOps = map[string]string{}
		p.keyOps = map[string]string{}
		for i, param := range method.TypeParams {
			p.zeroFactories[param] = "zero$" + param
			p.eqOps[param] = "eq$" + param
			p.cloneOps[param] = "clone$" + param
			p.setOps[param] = "set$" + param
			if i < len(method.KeyedParams) && method.KeyedParams[i] {
				p.keyOps[param] = "key$" + param
			}
		}
	}
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
	// Go's defer/panic drain: every deferred call runs at function exit,
	// LIFO; a panic from one defer replaces the in-flight panic but the
	// remaining (older) defers still run, and the last surviving panic
	// propagates. Normal returns run the same drain via finally.
	p.line("const _ds$: (() => void)[] = [];")
	p.line("let _dp$: { readonly v: unknown } | undefined = undefined;")
	p.line("try {")
	p.indent++
	if err := p.printBlockBody(body); err != nil {
		return err
	}
	p.indent--
	p.line("} catch (_de$) {")
	p.indent++
	p.line("_dp$ = { v: _de$ };")
	p.indent--
	p.line("} finally {")
	p.indent++
	p.line("for (let _di$ = _ds$.length - 1; _di$ >= 0; _di$--) {")
	p.indent++
	p.line("try {")
	p.indent++
	p.line("(_ds$[_di$] as () => void)();")
	p.indent--
	p.line("} catch (_de$) {")
	p.indent++
	p.line("_dp$ = { v: _de$ };")
	p.indent--
	p.line("}")
	p.indent--
	p.line("}")
	p.line("if (_dp$ !== undefined) { throw _dp$.v; }")
	p.indent--
	p.line("}")
	return nil
}

// eqComponent spells one field's exact equality comparison.
func (p *printer) eqComponent(left, right string, t ir.Type) (string, error) {
	if t.Kind == ir.KindIface && t.TypeParamName != "" {
		// A bare type-parameter field compares through the equality
		// operation in scope (the class's captured eq$P) — exact per
		// instantiation.
		op, has := p.eqOps[t.TypeParamName]
		if !has {
			return "", fmt.Errorf("no equality operation in scope for type parameter %q", t.TypeParamName)
		}
		return op + "(" + left + ", " + right + ")", nil
	}
	switch t.Kind {
	case ir.KindStruct:
		return left + ".goEq$(" + right + ")", nil
	case ir.KindIface:
		// An interface field compares through its own exact union equality
		// function (per-member narrowing), never an erased helper.
		union, err := p.tsType(t)
		if err != nil {
			return "", err
		}
		return union + "$eq(" + left + ", " + right + ")", nil
	case ir.KindArray:
		elem, err := p.eqComponent("$x", "$y", *t.Elem)
		if err != nil {
			return "", err
		}
		spelled, err := p.tsType(*t.Elem)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("gosl$.goArrayEqualWith(%s, %s, ($x: %s, $y: %s) => %s)", left, right, spelled, spelled, elem), nil
	}
	return "(" + left + " === " + right + ")", nil
}

// structKeyed reports whether the class captures key$P at position i.
func structKeyed(structDecl *ir.Struct, i int) bool {
	return i < len(structDecl.KeyedParams) && structDecl.KeyedParams[i]
}
