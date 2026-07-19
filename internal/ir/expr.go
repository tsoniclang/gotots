package ir

import (
	"fmt"
	"go/ast"
	"go/types"
)

// buildExpr converts one typed Go expression into IR, folding constants
// through go/constant so no textual constant parsing ever occurs.
func (b *builder) buildExpr(e ast.Expr) (Expr, error) {
	span := b.span(e.Pos())
	tv, ok := b.info.Types[e]
	if !ok {
		return nil, &Unsupported{Kind: KindExpressionWithoutTypeEvidence, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "expression without type evidence", Span: span}
	}

	// Untyped nil is only meaningful in a typed context (assignment,
	// argument, field, return, map value) or a comparison; those callers
	// use buildExprAs / the buildBinary intercept.
	if tv.IsNil() {
		return nil, &Unsupported{Kind: KindUntypedNilOutsideATypedContext, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "untyped nil outside a typed context", Span: span}
	}

	// Exact constant folding: any expression the Go type checker evaluated
	// to a constant becomes a Const with the exact value.
	if tv.Value != nil {
		t, err := b.typeOf(tv.Type, span)
		if err != nil {
			return nil, err
		}
		value, err := constValue(tv.Value, t, span)
		if err != nil {
			return nil, err
		}
		b.use("const")
		return &Const{T: t, Value: value}, nil
	}

	switch n := e.(type) {
	case *ast.ParenExpr:
		return b.buildExpr(n.X)

	case *ast.CompositeLit:
		t, err := b.typeOf(tv.Type, span)
		if err != nil {
			return nil, err
		}
		if t.Kind == KindSlice {
			out := &SliceLit{T: t}
			for _, element := range n.Elts {
				if _, isKeyed := element.(*ast.KeyValueExpr); isKeyed {
					return nil, &Unsupported{Kind: KindKeyedSliceLiteral, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "keyed slice literal", Span: span}
				}
				value, err := b.buildExprAs(element, *t.Elem)
				if err != nil {
					return nil, err
				}
				out.Values = append(out.Values, value)
			}
			b.use("sliceLiteral")
			return out, nil
		}
		if t.Kind == KindMap {
			out := &MapFrom{T: t}
			for _, element := range n.Elts {
				keyValue, ok := element.(*ast.KeyValueExpr)
				if !ok {
					return nil, &Unsupported{Kind: KindMapLiteralWithoutKeys, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "map literal without keys", Span: span}
				}
				key, err := b.buildExprAs(keyValue.Key, *t.Key)
				if err != nil {
					return nil, err
				}
				value, err := b.buildExprAs(keyValue.Value, *t.Elem)
				if err != nil {
					return nil, err
				}
				out.Keys = append(out.Keys, key)
				out.Values = append(out.Values, value)
			}
			b.use("mapLiteral")
			return out, nil
		}
		if t.Kind == KindStruct {
			return b.buildStructLit(n, t)
		}
		if t.Kind == KindArray {
			return b.buildArrayLit(n, t)
		}
		if t.Kind == KindUnit {
			// struct{}{} is the unit value.
			b.use("unitLiteral")
			return &Const{T: t, Value: "0"}, nil
		}
		if t.Kind == KindPointer && t.Elem != nil && t.Elem.Kind == KindStruct {
			// The elided allocation form inside composite literals:
			// []*T{{...}} allocates like &T{...}.
			return b.buildStructLit(n, t)
		}
		if t.Kind == KindExternal && len(n.Elts) == 0 {
			// T{} of an external type with no fields set is its zero
			// contract; keyed externals stay outside the reviewed surface.
			b.use("externZero")
			return &ExternZero{T: t}, nil
		}
		if t.Kind == KindExternal && len(n.Elts) > 0 {
			return b.buildExternLit(n, t, tv.Type, span)
		}
		return nil, &Unsupported{Kind: KindCompositeLiteralOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "composite literal of " + t.Go, Span: span}

	case *ast.FuncLit:
		return b.buildClosure(n)

	case *ast.Ident:
		object := b.info.Uses[n]
		if function, isFunc := object.(*types.Func); isFunc {
			return b.buildFuncRef(function, n, span)
		}
		variable, ok := object.(*types.Var)
		if !ok {
			return nil, &Unsupported{Kind: KindIdentifier, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("identifier %q (%T)", n.Name, object), Span: span}
		}
		return b.buildVarRef(variable, n.Name, span)

	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[n]
		if !ok {
			// A package-qualified identifier (no selection evidence).
			if base, isIdent := ast.Unparen(n.X).(*ast.Ident); isIdent {
				if _, isPkg := b.info.Uses[base].(*types.PkgName); isPkg {
					if function, isFunc := b.info.Uses[n.Sel].(*types.Func); isFunc {
						return b.buildFuncRef(function, n.Sel, span)
					}
					if variable, isVar := b.info.Uses[n.Sel].(*types.Var); isVar {
						return b.buildVarRef(variable, n.Sel.Name, span)
					}
				}
			}
			return nil, &Unsupported{Kind: KindNonFieldSelector, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-field selector " + n.Sel.Name + " (no selection evidence)", Span: span}
		}
		if selection.Kind() == types.MethodVal {
			return b.buildMethodValue(n, selection, tv.Type)
		}
		if selection.Kind() == types.MethodExpr {
			return b.buildMethodExpr(n, selection, tv.Type)
		}
		if selection.Kind() != types.FieldVal {
			return nil, &Unsupported{Kind: KindNonFieldSelector, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-field selector " + n.Sel.Name, Span: span}
		}
		base, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if len(selection.Index()) > 1 {
			// A promoted field: chain through the embedded fields.
			return b.chainFieldPath(base, b.info.Types[n.X].Type, selection.Index(), span)
		}
		if base.Type().Kind == KindExternal ||
			(base.Type().Kind == KindPointer && base.Type().Elem != nil && base.Type().Elem.Kind == KindExternal) {
			// An exported-field READ of an external struct value routes
			// through a typed per-field stub obligation.
			return b.buildExternFieldRead(base, n.Sel.Name, tv.Type, span)
		}
		if base.Type().Kind != KindPointer && base.Type().Kind != KindStruct {
			return nil, &Unsupported{Kind: KindFieldAccessOn, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "field access on " + base.Type().Go, Span: span}
		}
		t, err := b.typeOf(tv.Type, span)
		if err != nil {
			return nil, err
		}
		// Struct-valued field loads read the stored instance; Go's value
		// copy happens where the value is bound, never on the read path,
		// so addressable chains (x.F.G = v, x.F.M()) stay in place.
		b.use("fieldLoad")
		cell, err := b.fieldCell(selection, t)
		if err != nil {
			return nil, err
		}
		return &FieldLoad{X: base, Field: n.Sel.Name, T: t, Cell: cell}, nil

	case *ast.IndexExpr:
		// An EXPLICIT generic-function instantiation reference
		// (identity[int] as a value): the base ident carries the
		// instantiation evidence.
		if baseIdent, isIdent := ast.Unparen(n.X).(*ast.Ident); isIdent {
			if function, isFunc := b.info.Uses[baseIdent].(*types.Func); isFunc {
				return b.buildFuncRef(function, baseIdent, span)
			}
		}
		if selector, isSel := ast.Unparen(n.X).(*ast.SelectorExpr); isSel {
			if function, isFunc := b.info.Uses[selector.Sel].(*types.Func); isFunc {
				return b.buildFuncRef(function, selector.Sel, span)
			}
		}
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		switch operand.Type().Kind {
		case KindMap:
			key, err := b.buildExprAs(n.Index, *operand.Type().Key)
			if err != nil {
				return nil, err
			}
			b.use("mapGet")
			return &MapGet{Map: operand, Key: key, T: *operand.Type().Elem}, nil
		case KindSlice:
			index, err := b.buildExpr(n.Index)
			if err != nil {
				return nil, err
			}
			b.use("sliceGet")
			return &SliceGet{X: operand, Index: index, T: *operand.Type().Elem}, nil
		case KindArray:
			index, err := b.buildExpr(n.Index)
			if err != nil {
				return nil, err
			}
			b.use("arrayGet")
			return &ArrayGet{X: operand, Index: index, T: *operand.Type().Elem}, nil
		case KindString:
			return b.buildStringIndex(operand, n.Index)
		case KindPointer:
			if operand.Type().Elem != nil && operand.Type().Elem.Kind == KindArray {
				// Go's implicit (*p)[i]: deref (nil panics), then index.
				deref := &Deref{X: operand, T: *operand.Type().Elem}
				index, err := b.buildExpr(n.Index)
				if err != nil {
					return nil, err
				}
				b.use("arrayGet")
				return &ArrayGet{X: deref, Index: index, T: *deref.T.Elem}, nil
			}
		}
		return nil, &Unsupported{Kind: KindIndexOn, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "index on " + operand.Type().Go, Span: span}

	case *ast.SliceExpr:
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if n.Slice3 {
			// The three-index form s[low:high:max] caps the result at
			// max-low so a later append reallocates sooner, protecting the
			// operand's tail. It is defined only over slices (Go forbids it
			// on strings; array operands would need the array-view carrier).
			if operand.Type().Kind != KindSlice {
				return nil, &Unsupported{Kind: KindFullSliceExpressionOn, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "full slice expression on " + operand.Type().Go, Span: span}
			}
			out := &SliceReslice{X: operand, T: operand.Type()}
			if n.Low != nil {
				if out.Low, err = b.buildExpr(n.Low); err != nil {
					return nil, err
				}
			}
			if out.High, err = b.buildExpr(n.High); err != nil {
				return nil, err
			}
			if out.Max, err = b.buildExpr(n.Max); err != nil {
				return nil, err
			}
			b.use("reslice3")
			return out, nil
		}
		if operand.Type().Kind == KindArray {
			// Slicing an addressable array (the checker enforces
			// addressability) yields a slice sharing its storage.
			view := &ArraySliceView{X: operand, T: Type{Kind: KindSlice, Go: "[]" + operand.Type().Elem.Go, Elem: operand.Type().Elem}}
			if n.Low != nil {
				if view.Low, err = b.buildExpr(n.Low); err != nil {
					return nil, err
				}
			}
			if n.High != nil {
				if view.High, err = b.buildExpr(n.High); err != nil {
					return nil, err
				}
			}
			b.use("arraySliceView")
			return view, nil
		}
		if operand.Type().Kind == KindString {
			return b.buildStringSlice(operand, n)
		}
		if operand.Type().Kind != KindSlice {
			return nil, &Unsupported{Kind: KindResliceOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reslice of " + operand.Type().Go, Span: span}
		}
		out := &SliceReslice{X: operand, T: operand.Type()}
		if n.Low != nil {
			low, err := b.buildExpr(n.Low)
			if err != nil {
				return nil, err
			}
			out.Low = low
		}
		if n.High != nil {
			high, err := b.buildExpr(n.High)
			if err != nil {
				return nil, err
			}
			out.High = high
		}
		b.use("reslice")
		return out, nil

	case *ast.UnaryExpr:
		return b.buildUnary(n, tv.Type)

	case *ast.StarExpr:
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind != KindPointer || operand.Type().Elem == nil {
			return nil, &Unsupported{Kind: KindDereferenceOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "dereference of " + operand.Type().Go, Span: span}
		}
		if operand.Type().Elem.Kind == KindIface && operand.Type().Elem.TypeParamName != "" {
			if cell, isCell := operand.(*CellNew); isCell {
				// The *new(T) zero idiom: each evaluation is a fresh zero
				// of the binding — exactly the cell's zero expression.
				b.use("paramZeroDeref")
				return cell.Zero, nil
			}
			if b.declPtrRequires(operand.Type().Elem.TypeParamName) {
				// A pointer-split declaration: each family emission derefs
				// its exact form.
				b.use("deref")
				return &Deref{X: operand, T: *operand.Type().Elem}, nil
			}
			// The GoPtr carrier is opaque inside the generic body: reading
			// through it needs a per-instantiation operation the protocol
			// does not yet carry — fail closed to a typed placeholder.
			return nil, &Unsupported{Kind: KindDereferenceOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "dereference of " + operand.Type().Go, Span: span}
		}
		b.use("deref")
		return &Deref{X: operand, T: *operand.Type().Elem}, nil

	case *ast.TypeAssertExpr:
		return b.buildTypeAssert(n, false)

	case *ast.BinaryExpr:
		return b.buildBinary(n, tv.Type)

	case *ast.CallExpr:
		if tv.IsType() {
			return nil, &Unsupported{Kind: KindTypeInCallPosition, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "type in call position", Span: span}
		}
		if convert, is := b.conversionTarget(n); is {
			to, err := b.typeOf(convert, span)
			if err != nil {
				return nil, err
			}
			if argInfo, hasArg := b.info.Types[n.Args[0]]; hasArg && argInfo.IsNil() {
				// T(nil): the typed nil of a nilable target.
				if !to.Kind.Nilable() {
					return nil, &Unsupported{Kind: KindConversionFromUntypedNilTo, Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "conversion from untyped nil to " + to.Go, Span: span}
				}
				b.use("convert")
				return &NilConst{T: to}, nil
			}
			x, err := b.buildExpr(n.Args[0])
			if err != nil {
				return nil, err
			}
			if converted, isString := b.buildStringConversion(x, to); isString {
				return converted, nil
			}
			if bridged, isBridge, err := b.buildExternOwnedConversion(x, to, b.info.Types[n.Args[0]].Type, convert, span); isBridge {
				return bridged, err
			}
			if x.Type().Kind == to.Kind &&
				types.Identical(types.Unalias(b.info.Types[n.Args[0]].Type).Underlying(), types.Unalias(convert).Underlying()) {
				switch to.Kind {
				case KindMap, KindSlice, KindFunc, KindChan, KindStruct, KindArray, KindPointer:
					// Same-underlying conversion between named forms: the
					// value IS the value (Go's conversion copies at BIND
					// sites exactly as assignment does).
					b.use("convert")
					return &Convert{X: x, To: to}, nil
				}
			}
			if x.Type().Kind == KindPointer && to.Kind == KindPointer &&
				sameUnderlyingNamedPointers(b.info.Types[n.Args[0]].Type, convert) {
				// Same-underlying owned pointer conversion (type
				// MutableNode Node): the instance IS the value — identity
				// at the carrier, the classes are structurally identical.
				b.use("convert")
				return &Convert{X: x, To: to}, nil
			}
			fromT := x.Type()
			if to.TypeParamName != "" && to.ParamRepr != nil && to.ParamReprExact {
				// Conversion INTO an exact-kind uniform parameter: convert
				// to the representative, view the result as the parameter.
				source := fromT
				if source.TypeParamName != "" && source.ParamRepr != nil {
					x = &ParamReprView{X: x, T: *source.ParamRepr}
					source = *source.ParamRepr
				}
				if err := b.checkConversion(source, *to.ParamRepr, span); err != nil {
					return nil, err
				}
				b.use("param-repr")
				return &ParamReprCast{X: &Convert{X: x, To: *to.ParamRepr}, T: to}, nil
			}
			if fromT.TypeParamName != "" && fromT.ParamRepr != nil && to.TypeParamName == "" {
				// Conversion OUT of a carrier-uniform parameter: the wrap is
				// a function of the target alone on the shared carrier.
				if err := b.checkConversion(*fromT.ParamRepr, to, span); err != nil {
					return nil, err
				}
				b.use("param-repr")
				return &Convert{X: &ParamReprView{X: x, T: *fromT.ParamRepr}, To: to}, nil
			}
			if err := b.checkConversion(x.Type(), to, span); err != nil {
				return nil, err
			}
			b.use("convert")
			return &Convert{X: x, To: to}, nil
		}
		if builtin, isBuiltin := b.builtinCallee(n); isBuiltin {
			return b.buildBuiltin(n, builtin, tv.Type)
		}
		call, err := b.buildAnyCall(n)
		if err != nil {
			return nil, err
		}
		if _, isTuple := b.info.Types[n].Type.(*types.Tuple); isTuple {
			return nil, &Unsupported{Kind: KindMultiResultCallInExpressionPosition, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "multi-result call in expression position", Span: span}
		}
		return call, nil
	}
	return nil, &Unsupported{Kind: KindUnrecognizedExpression, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "unrecognized expression " + fmt.Sprintf("%T", e), Span: span}
}

// buildExprAs builds an expression whose context expects a known type,
// giving untyped nil its exact typed zero and inserting Go's value copy
// when a struct value is bound.
func (b *builder) buildExprAs(e ast.Expr, expected Type) (Expr, error) {
	if tv, ok := b.info.Types[e]; ok && tv.IsNil() {
		if expected.Kind.Nilable() {
			b.use("nil")
			return &NilConst{T: expected}, nil
		}
		return nil, &Unsupported{Kind: KindNilOfType, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "nil of type " + expected.Go, Span: b.span(e.Pos())}
	}
	built, err := b.buildExpr(e)
	if err != nil {
		return nil, err
	}
	if expected.Kind == KindStruct || expected.Kind == KindArray || expected.Kind == KindExternal {
		return b.bindStructValue(built), nil
	}
	if expected.Kind == KindIface && expected.TypeParamName != "" {
		return b.bindParamValue(built, expected.TypeParamName), nil
	}
	if expected.Kind == KindIface {
		return b.boxIfaceValue(built, b.info.Types[e].Type, expected, b.span(e.Pos()))
	}
	return built, nil
}

// bindParamValue inserts the per-binding value copy of a type-parameter
// bind site: loads wrap in ParamCopy (the clone$P factory copies exactly
// for value-copy carriers and is the identity otherwise); already-fresh
// values — allocations, zeros, copies, call results — bind directly.
func (b *builder) bindParamValue(e Expr, param string) Expr {
	switch e.(type) {
	case *StructNew, *StructZero, *StructCopy, *ParamCopy, *ParamZero, *Call, *MethodCall, *ArrayLit, *ExternZero, *ExternalMethodCall, *ExternalCall:
		return e
	}
	b.use("paramCopy")
	return &ParamCopy{X: e, Param: param}
}

// buildVarRef reads a variable. Package-level variables carry their
// declaring package: reads from other unit packages go through the live
// ESM namespace binding; a variable outside the unit fails closed.
func (b *builder) buildVarRef(variable *types.Var, name string, span Span) (Expr, error) {
	t, err := b.typeOf(variable.Type(), span)
	if err != nil {
		return nil, err
	}
	pkg := ""
	if variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
		if !b.unit.Owns(variable.Pkg().Path()) {
			// An external package variable reads through its typed stub:
			// the assembled implementation supplies the identity-stable
			// value; writes stay outside the reviewed surface.
			b.use("externVar")
			id := variable.Pkg().Path() + "." + variable.Name()
			b.unit.AddExternalVar(id, variable)
			return &ExternVar{ID: id, T: t}, nil
		}
		pkg = variable.Pkg().Path()
	}
	if pkg == "" && b.boxed[variable] && boxable(t.Kind) {
		b.use("boxedLoad")
		return &BoxedLoad{Cell: cellName(b.bindNameVar(variable, name)), T: t}, nil
	}
	b.use("varRef")
	if pkg == "" {
		name = b.bindNameVar(variable, name)
	}
	return &VarRef{Name: name, Pkg: pkg, T: t}, nil
}
