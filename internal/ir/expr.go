package ir

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
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
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "composite literal of " + t.Go + " (struct values need &T{...})", Span: span}

	case *ast.Ident:
		object := b.info.Uses[n]
		variable, ok := object.(*types.Var)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("identifier %q (%T)", n.Name, object), Span: span}
		}
		t, err := b.typeOf(variable.Type(), span)
		if err != nil {
			return nil, err
		}
		if t.Kind == KindStruct {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct value variable (value-copy semantics)", Span: span}
		}
		b.use("varRef")
		return &VarRef{Name: n.Name, T: t}, nil

	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[n]
		if !ok || selection.Kind() != types.FieldVal {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-field selector", Span: span}
		}
		base, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if base.Type().Kind != KindPointer {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "field access on " + base.Type().Go, Span: span}
		}
		t, err := b.typeOf(tv.Type, span)
		if err != nil {
			return nil, err
		}
		if t.Kind == KindStruct {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct-valued field load (value-copy semantics)", Span: span}
		}
		b.use("fieldLoad")
		return &FieldLoad{X: base, Field: n.Sel.Name, T: t}, nil

	case *ast.IndexExpr:
		mapExpr, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if mapExpr.Type().Kind != KindMap {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "index on " + mapExpr.Type().Go, Span: span}
		}
		key, err := b.buildExpr(n.Index)
		if err != nil {
			return nil, err
		}
		b.use("mapGet")
		return &MapGet{Map: mapExpr, Key: key, T: *mapExpr.Type().Elem}, nil

	case *ast.UnaryExpr:
		return b.buildUnary(n, tv.Type)

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
// giving untyped nil its exact typed zero.
func (b *builder) buildExprAs(e ast.Expr, expected Type) (Expr, error) {
	if tv, ok := b.info.Types[e]; ok && tv.IsNil() {
		if expected.Kind == KindPointer || expected.Kind == KindMap {
			b.use("nil")
			return &NilConst{T: expected}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "nil of type " + expected.Go, Span: b.span(e.Pos())}
	}
	return b.buildExpr(e)
}

func (b *builder) buildUnary(n *ast.UnaryExpr, resultType types.Type) (Expr, error) {
	span := b.span(n.Pos())
	switch n.Op {
	case token.AND:
		// Heap allocation of a struct literal: &T{...}.
		lit, ok := ast.Unparen(n.X).(*ast.CompositeLit)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "address of non-composite-literal", Span: span}
		}
		return b.buildStructNew(lit, resultType)
	case token.SUB, token.NOT, token.XOR, token.ADD:
		x, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if n.Op == token.ADD {
			return x, nil // unary plus is identity
		}
		t, err := b.typeOf(resultType, span)
		if err != nil {
			return nil, err
		}
		b.use("unary:" + n.Op.String())
		return &Unary{Op: n.Op, X: x, T: t}, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "unary operator " + n.Op.String(), Span: span}
}

// buildStructNew lowers &T{...} into an ordered full-field allocation.
func (b *builder) buildStructNew(lit *ast.CompositeLit, resultType types.Type) (Expr, error) {
	span := b.span(lit.Pos())
	t, err := b.typeOf(resultType, span)
	if err != nil {
		return nil, err
	}
	if t.Kind != KindPointer {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "composite literal of " + t.Go, Span: span}
	}
	named := types.Unalias(b.info.Types[lit].Type).(*types.Named)
	structType := named.Underlying().(*types.Struct)

	fieldIRType := func(field *types.Var) (Type, error) { return b.typeOf(field.Type(), span) }
	fieldByName := map[string]*types.Var{}
	for i := range structType.NumFields() {
		fieldByName[structType.Field(i).Name()] = structType.Field(i)
	}

	// Resolve provided values, keyed or positional, typing each against
	// its field so nil literals get exact zeros.
	provided := map[string]Expr{}
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
			}
		}
	}

	out := &StructNew{TypeName: t.Named, T: t}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		if value, ok := provided[field.Name()]; ok {
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
	b.use("structNew")
	return out, nil
}

// builtinCallee resolves a call to a predeclared operation by object
// identity, never spelling.
func (b *builder) builtinCallee(call *ast.CallExpr) (*types.Builtin, bool) {
	ident, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok {
		return nil, false
	}
	builtin, ok := b.info.Uses[ident].(*types.Builtin)
	return builtin, ok
}

// buildBuiltin lowers the reviewed predeclared operations.
func (b *builder) buildBuiltin(call *ast.CallExpr, builtin *types.Builtin, resultType types.Type) (Expr, error) {
	span := b.span(call.Pos())
	switch builtin.Name() {
	case "len":
		operand, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		switch operand.Type().Kind {
		case KindString:
			b.use("len:string")
			return &StringLen{X: operand}, nil
		case KindMap:
			b.use("len:map")
			return &MapLen{X: operand}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "len of " + operand.Type().Go, Span: span}
	case "make":
		t, err := b.typeOf(resultType, span)
		if err != nil {
			return nil, err
		}
		if t.Kind == KindMap && len(call.Args) == 1 {
			b.use("makeMap")
			return &MapMake{T: t}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "make of " + t.Go, Span: span}
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "builtin " + builtin.Name(), Span: span}
}

// buildAnyCall lowers a direct function call or a pointer-receiver method
// call within the translated package.
func (b *builder) buildAnyCall(n *ast.CallExpr) (Expr, error) {
	span := b.span(n.Pos())
	if n.Ellipsis.IsValid() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "slice-expansion call", Span: span}
	}

	// Method call through selection evidence.
	if selector, ok := ast.Unparen(n.Fun).(*ast.SelectorExpr); ok {
		selection, hasSelection := b.info.Selections[selector]
		if hasSelection && selection.Kind() == types.MethodVal {
			return b.buildMethodCall(n, selector, selection)
		}
	}

	callee, ok := ast.Unparen(n.Fun).(*ast.Ident)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", n.Fun), Span: span}
	}
	object := b.info.Uses[callee]
	function, ok := object.(*types.Func)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", object), Span: span}
	}
	if function.Pkg() == nil || function.Pkg().Path() != b.pkgPath {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call outside the translated package", Span: span}
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() != nil || signature.Variadic() || signature.TypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "variadic or generic call", Span: span}
	}

	call := &Call{Callee: function.Name()}
	if err := b.buildCallArgsResults(n, signature, &call.Args, &call.Results); err != nil {
		return nil, err
	}
	b.use("call")
	return call, nil
}

func (b *builder) buildMethodCall(n *ast.CallExpr, selector *ast.SelectorExpr, selection *types.Selection) (Expr, error) {
	span := b.span(n.Pos())
	method := selection.Obj().(*types.Func)
	if method.Pkg() == nil || method.Pkg().Path() != b.pkgPath {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call outside the translated package", Span: span}
	}
	recv, err := b.buildExpr(selector.X)
	if err != nil {
		return nil, err
	}
	if recv.Type().Kind != KindPointer {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call on " + recv.Type().Go, Span: span}
	}
	signature := method.Type().(*types.Signature)
	if signature.Variadic() || signature.TypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "variadic or generic method call", Span: span}
	}
	if _, isPointerRecv := signature.Recv().Type().(*types.Pointer); !isPointerRecv {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "value-receiver method call (receiver copy semantics)", Span: span}
	}

	out := &MethodCall{Recv: recv, Method: method.Name()}
	if err := b.buildCallArgsResults(n, signature, &out.Args, &out.Results); err != nil {
		return nil, err
	}
	b.use("methodCall")
	return out, nil
}

func (b *builder) buildCallArgsResults(n *ast.CallExpr, signature *types.Signature, args *[]Expr, results *[]Type) error {
	span := b.span(n.Pos())
	for i, arg := range n.Args {
		expected, err := b.typeOf(signature.Params().At(i).Type(), b.span(arg.Pos()))
		if err != nil {
			return err
		}
		built, err := b.buildExprAs(arg, expected)
		if err != nil {
			return err
		}
		*args = append(*args, built)
	}
	tuple := signature.Results()
	for i := range tuple.Len() {
		t, err := b.typeOf(tuple.At(i).Type(), span)
		if err != nil {
			return err
		}
		*results = append(*results, t)
	}
	return nil
}

// conversionTarget reports whether the call is a type conversion and, if
// so, the target type — decided by type-checker evidence, never spelling.
func (b *builder) conversionTarget(call *ast.CallExpr) (types.Type, bool) {
	if tv, ok := b.info.Types[call.Fun]; ok && tv.IsType() && len(call.Args) == 1 {
		return tv.Type, true
	}
	return nil, false
}

// checkConversion admits only conversions in the reviewed subset:
// integer-to-integer of any widths, and integer-to-float64.
func (b *builder) checkConversion(from, to Type, span Span) error {
	if from.Kind.Integer() && to.Kind.Integer() {
		return nil
	}
	if from.Kind.Integer() && to.Kind == KindFloat64 {
		return nil
	}
	return &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION",
		Construct: "conversion from " + from.Go + " to " + to.Go, Span: span}
}

// constValue renders an exact go/constant value for the resolved type.
func constValue(v constant.Value, t Type, span Span) (string, error) {
	switch t.Kind {
	case KindBool:
		return fmt.Sprintf("%v", constant.BoolVal(v)), nil
	case KindString:
		quoted, err := json.Marshal(constant.StringVal(v))
		if err != nil {
			return "", err
		}
		return string(quoted), nil
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
	case t.Kind == KindPointer, t.Kind == KindMap:
		return &NilConst{T: t}, nil
	case t.Kind.Integer(), t.Kind.Float():
		return &Const{T: t, Value: "0"}, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "zero value of " + t.Go, Span: span}
}
