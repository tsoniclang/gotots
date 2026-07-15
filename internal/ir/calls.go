package ir

import (
	"fmt"
	"go/ast"
	"go/types"
)

// buildAnyCall lowers a direct function call or a pointer-receiver method
// call within the translated unit.
func (b *builder) buildAnyCall(n *ast.CallExpr) (Expr, error) {
	span := b.span(n.Pos())

	// The callee is a plain identifier, a method selection, or a
	// package-qualified identifier (pkg.Fn — qualified identifiers have no
	// selection evidence; their object comes from Uses on the selector).
	var object types.Object
	switch fun := ast.Unparen(n.Fun).(type) {
	case *ast.SelectorExpr:
		if selection, hasSelection := b.info.Selections[fun]; hasSelection {
			if selection.Kind() == types.MethodVal {
				return b.buildMethodCall(n, fun, selection)
			}
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call of selected value", Span: span}
		}
		base, isIdent := ast.Unparen(fun.X).(*ast.Ident)
		if !isIdent {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", n.Fun), Span: span}
		}
		if _, isPkg := b.info.Uses[base].(*types.PkgName); !isPkg {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", n.Fun), Span: span}
		}
		object = b.info.Uses[fun.Sel]
	case *ast.Ident:
		object = b.info.Uses[fun]
	default:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", n.Fun), Span: span}
	}

	function, ok := object.(*types.Func)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", object), Span: span}
	}
	if function.Pkg() == nil || !b.unit[function.Pkg().Path()] {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call outside the translated unit", Span: span}
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() != nil || signature.TypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic call", Span: span}
	}

	call := &Call{Pkg: function.Pkg().Path(), Callee: function.Name()}
	if err := b.buildCallArgsResults(n, signature, &call.Args, &call.Results); err != nil {
		return nil, err
	}
	b.use("call")
	return call, nil
}

func (b *builder) buildMethodCall(n *ast.CallExpr, selector *ast.SelectorExpr, selection *types.Selection) (Expr, error) {
	span := b.span(n.Pos())
	method := selection.Obj().(*types.Func)
	if method.Pkg() == nil || !b.unit[method.Pkg().Path()] {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call outside the translated unit", Span: span}
	}
	recv, err := b.buildExpr(selector.X)
	if err != nil {
		return nil, err
	}
	if recv.Type().Kind != KindPointer && recv.Type().Kind != KindStruct {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call on " + recv.Type().Go, Span: span}
	}
	signature := method.Type().(*types.Signature)
	if signature.TypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic method call", Span: span}
	}
	// Receiver flavors are exact through the generated class: a value
	// receiver clones on method entry (the caller's instance is never
	// mutated), and a pointer receiver binds the instance itself — the
	// type checker has already required an addressable receiver, whose
	// stores this lowering keeps in place.

	out := &MethodCall{Recv: recv, Method: method.Name()}
	if err := b.buildCallArgsResults(n, signature, &out.Args, &out.Results); err != nil {
		return nil, err
	}
	b.use("methodCall")
	return out, nil
}

func (b *builder) buildCallArgsResults(n *ast.CallExpr, signature *types.Signature, args *[]Expr, results *[]Type) error {
	span := b.span(n.Pos())
	params := signature.Params()
	fixed := params.Len()
	if signature.Variadic() {
		fixed--
	}
	for i := 0; i < fixed; i++ {
		expected, err := b.typeOf(params.At(i).Type(), b.span(n.Args[i].Pos()))
		if err != nil {
			return err
		}
		built, err := b.buildExprAs(n.Args[i], expected)
		if err != nil {
			return err
		}
		*args = append(*args, built)
	}
	if signature.Variadic() {
		variadic, err := b.buildVariadicSlot(n, params.At(fixed), fixed)
		if err != nil {
			return err
		}
		*args = append(*args, variadic)
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

// buildVariadicSlot materializes the final argument of a variadic call
// with Go's exact packing: a spread call (xs...) passes the very slice
// value (aliasing preserved), no trailing arguments pass nil, and
// trailing arguments allocate a fresh slice per call.
func (b *builder) buildVariadicSlot(n *ast.CallExpr, param *types.Var, fixed int) (Expr, error) {
	sliceType, err := b.typeOf(param.Type(), b.span(n.Pos()))
	if err != nil {
		return nil, err
	}
	if n.Ellipsis.IsValid() {
		b.use("variadic:spread")
		return b.buildExprAs(n.Args[len(n.Args)-1], sliceType)
	}
	if len(n.Args) == fixed {
		b.use("variadic:empty")
		return &NilConst{T: sliceType}, nil
	}
	out := &SliceLit{T: sliceType}
	for _, arg := range n.Args[fixed:] {
		built, err := b.buildExprAs(arg, *sliceType.Elem)
		if err != nil {
			return nil, err
		}
		out.Values = append(out.Values, built)
	}
	b.use("variadic:pack")
	return out, nil
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
