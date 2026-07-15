package ir

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"math/big"
	"strconv"
)

// buildExpr converts one typed Go expression into IR, folding constants
// through go/constant so no textual constant parsing ever occurs.
func (b *builder) buildExpr(e ast.Expr) (Expr, error) {
	span := b.span(e.Pos())
	tv, ok := b.info.Types[e]
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "expression without type evidence", Span: span}
	}

	// Untyped nil is only meaningful in a typed context (assignment,
	// argument, field, return, map value) or a comparison; those callers
	// use buildExprAs / the buildBinary intercept.
	if tv.IsNil() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "untyped nil outside a typed context", Span: span}
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
					return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "keyed slice literal", Span: span}
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
					return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "map literal without keys", Span: span}
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
		if t.Kind == KindExternal && len(n.Elts) == 0 {
			// T{} of an external type with no fields set is its zero
			// contract; keyed externals stay outside the reviewed surface.
			b.use("externZero")
			return &ExternZero{T: t}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "composite literal of " + t.Go, Span: span}

	case *ast.FuncLit:
		return b.buildClosure(n)

	case *ast.Ident:
		object := b.info.Uses[n]
		if function, isFunc := object.(*types.Func); isFunc {
			return b.buildFuncRef(function, span)
		}
		variable, ok := object.(*types.Var)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("identifier %q (%T)", n.Name, object), Span: span}
		}
		return b.buildVarRef(variable, n.Name, span)

	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[n]
		if !ok {
			// A package-qualified identifier (no selection evidence).
			if base, isIdent := ast.Unparen(n.X).(*ast.Ident); isIdent {
				if _, isPkg := b.info.Uses[base].(*types.PkgName); isPkg {
					if function, isFunc := b.info.Uses[n.Sel].(*types.Func); isFunc {
						return b.buildFuncRef(function, span)
					}
					if variable, isVar := b.info.Uses[n.Sel].(*types.Var); isVar {
						return b.buildVarRef(variable, n.Sel.Name, span)
					}
				}
			}
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-field selector " + n.Sel.Name + " (no selection evidence)", Span: span}
		}
		if selection.Kind() == types.MethodVal {
			return b.buildMethodValue(n, selection, tv.Type)
		}
		if selection.Kind() == types.MethodExpr {
			return b.buildMethodExpr(n, selection, tv.Type)
		}
		if selection.Kind() != types.FieldVal {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-field selector " + n.Sel.Name, Span: span}
		}
		base, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if len(selection.Index()) > 1 {
			// A promoted field: chain through the embedded fields.
			return b.chainFieldPath(base, b.info.Types[n.X].Type, selection.Index(), span)
		}
		if base.Type().Kind != KindPointer && base.Type().Kind != KindStruct {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "field access on " + base.Type().Go, Span: span}
		}
		t, err := b.typeOf(tv.Type, span)
		if err != nil {
			return nil, err
		}
		// Struct-valued field loads read the stored instance; Go's value
		// copy happens where the value is bound, never on the read path,
		// so addressable chains (x.F.G = v, x.F.M()) stay in place.
		b.use("fieldLoad")
		return &FieldLoad{X: base, Field: n.Sel.Name, T: t}, nil

	case *ast.IndexExpr:
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
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "index on " + operand.Type().Go, Span: span}

	case *ast.SliceExpr:
		if n.Slice3 {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "full slice expression (capacity-limiting)", Span: span}
		}
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
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
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reslice of " + operand.Type().Go, Span: span}
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
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "dereference of " + operand.Type().Go, Span: span}
		}
		b.use("deref")
		return &Deref{X: operand, T: *operand.Type().Elem}, nil

	case *ast.TypeAssertExpr:
		return b.buildTypeAssert(n, false)

	case *ast.BinaryExpr:
		return b.buildBinary(n, tv.Type)

	case *ast.CallExpr:
		if tv.IsType() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "type in call position", Span: span}
		}
		if convert, is := b.conversionTarget(n); is {
			x, err := b.buildExpr(n.Args[0])
			if err != nil {
				return nil, err
			}
			to, err := b.typeOf(convert, span)
			if err != nil {
				return nil, err
			}
			if converted, isString := b.buildStringConversion(x, to); isString {
				return converted, nil
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
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "multi-result call in expression position", Span: span}
		}
		return call, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("%T", e), Span: span}
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
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "nil of type " + expected.Go, Span: b.span(e.Pos())}
	}
	built, err := b.buildExpr(e)
	if err != nil {
		return nil, err
	}
	if expected.Kind == KindStruct || expected.Kind == KindArray || expected.Kind == KindExternal {
		return b.bindStructValue(built), nil
	}
	if expected.Kind == KindIface {
		return b.boxIfaceValue(built, b.info.Types[e].Type, expected, b.span(e.Pos()))
	}
	return built, nil
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
			// An external package variable reads through its contract:
			// the registered behavior supplies the (identity-stable)
			// value; writes stay outside the reviewed surface.
			b.use("externVar")
			return &ExternVar{ID: variable.Pkg().Path() + "." + variable.Name(), T: t}, nil
		}
		pkg = variable.Pkg().Path()
	}
	if pkg == "" && b.boxed[variable] && boxable(t.Kind) {
		b.use("boxedLoad")
		return &BoxedLoad{Cell: cellName(name), T: t}, nil
	}
	b.use("varRef")
	return &VarRef{Name: name, Pkg: pkg, T: t}, nil
}

// bindStructValue inserts the value copy of a struct binding. Loads from
// variables, fields, and elements clone; already-fresh values — new
// allocations, zeros, copies, and call results (copied at their return
// sites) — bind directly.
func (b *builder) bindStructValue(e Expr) Expr {
	kind := e.Type().Kind
	if kind != KindStruct && kind != KindArray && kind != KindExternal {
		return e
	}
	switch e.(type) {
	case *StructNew, *StructZero, *StructCopy, *Call, *MethodCall, *ArrayLit, *ExternZero, *ExternalMethodCall, *ExternalCall:
		return e
	}
	switch kind {
	case KindArray:
		b.use("arrayCopy")
	case KindExternal:
		b.use("externCopy")
	default:
		b.use("structCopy")
	}
	return &StructCopy{X: e}
}

// buildStructLit materializes a struct composite literal — as a heap
// allocation (&T{...}, pointer type) or a fresh struct value (T{...}) —
// with every field value in declaration order (omitted fields are
// explicit zeros).
func (b *builder) buildStructLit(lit *ast.CompositeLit, t Type) (Expr, error) {
	span := b.span(lit.Pos())
	named := types.Unalias(b.info.Types[lit].Type).(*types.Named)
	structType := named.Underlying().(*types.Struct)

	fieldIRType := func(field *types.Var) (Type, error) { return b.typeOf(field.Type(), span) }
	fieldByName := map[string]*types.Var{}
	for i := range structType.NumFields() {
		fieldByName[structType.Field(i).Name()] = structType.Field(i)
	}

	// Resolve provided values, keyed or positional, typing each against
	// its field so nil literals get exact zeros. Source order is
	// recorded: Go evaluates literal values in the order written.
	provided := map[string]Expr{}
	var sourceOrder []string
	if len(lit.Elts) > 0 {
		_, keyed := lit.Elts[0].(*ast.KeyValueExpr)
		for index, element := range lit.Elts {
			if keyValue, isKeyed := element.(*ast.KeyValueExpr); isKeyed != keyed {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "mixed keyed and positional literal", Span: span}
			} else if isKeyed {
				name := keyValue.Key.(*ast.Ident).Name
				expected, err := fieldIRType(fieldByName[name])
				if err != nil {
					return nil, err
				}
				value, err := b.buildExprAs(keyValue.Value, expected)
				if err != nil {
					return nil, err
				}
				provided[name] = value
				sourceOrder = append(sourceOrder, name)
			} else {
				expected, err := fieldIRType(structType.Field(index))
				if err != nil {
					return nil, err
				}
				value, err := b.buildExprAs(element, expected)
				if err != nil {
					return nil, err
				}
				provided[structType.Field(index).Name()] = value
				sourceOrder = append(sourceOrder, structType.Field(index).Name())
			}
		}
	}

	out := &StructNew{Pkg: t.Pkg, TypeName: t.Named, T: t}
	argIndexOf := map[string]int{}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		if value, ok := provided[field.Name()]; ok {
			argIndexOf[field.Name()] = len(out.Args)
			out.Args = append(out.Args, value)
			continue
		}
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		zero, err := zeroValue(fieldType, span)
		if err != nil {
			return nil, err
		}
		out.Args = append(out.Args, zero)
	}
	// A keyed literal whose source order differs from field order stages
	// its provided values so they evaluate in source order.
	ascending := true
	order := make([]int, 0, len(sourceOrder))
	for _, name := range sourceOrder {
		order = append(order, argIndexOf[name])
	}
	for k := 1; k < len(order); k++ {
		if order[k] < order[k-1] {
			ascending = false
			break
		}
	}
	if !ascending {
		out.EvalOrder = order
		b.use("structLitStaged")
	}
	b.use("structNew")
	return out, nil
}

// constValue renders an exact go/constant value for the resolved type.
func constValue(v constant.Value, t Type, span Span) (string, error) {
	switch t.Kind {
	case KindBool:
		return fmt.Sprintf("%v", constant.BoolVal(v)), nil
	case KindString:
		// The byte-string carrier holds one Go byte per JS code unit, so
		// the literal spells each byte: printable ASCII directly, all
		// other bytes (including non-UTF-8) as \xNN escapes.
		return byteStringLiteral(constant.StringVal(v)), nil
	case KindFloat32, KindFloat64:
		f, _ := constant.Float64Val(v)
		return strconv.FormatFloat(f, 'g', -1, 64), nil
	default:
		if !t.Kind.Integer() {
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "constant of type " + t.Go, Span: span}
		}
		text := v.ExactString()
		if _, ok := new(big.Int).SetString(text, 10); !ok {
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-integral integer constant " + text, Span: span}
		}
		return text, nil
	}
}

// zeroValue materializes the Go zero value for a reviewed type.
func zeroValue(t Type, span Span) (Expr, error) {
	switch {
	case t.Kind == KindBool:
		return &Const{T: t, Value: "false"}, nil
	case t.Kind == KindString:
		return &Const{T: t, Value: `""`}, nil
	case t.Kind.Nilable():
		return &NilConst{T: t}, nil
	case t.Kind == KindStruct, t.Kind == KindArray:
		return &StructZero{T: t}, nil
	case t.Kind == KindUnit:
		return &Const{T: t, Value: "0"}, nil
	case t.Kind == KindExternal:
		return &ExternZero{T: t}, nil
	case t.Kind.Integer(), t.Kind.Float():
		return &Const{T: t, Value: "0"}, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "zero value of " + t.Go, Span: span}
}
