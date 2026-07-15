package ir

import (
	"go/ast"
	"go/types"
)

// buildAnyCall lowers a call: a statically resolved package-level
// function, a statically resolved method, or a dynamic call of any
// function-typed value (a nil value panics at the call, as in Go).
func (b *builder) buildAnyCall(n *ast.CallExpr) (Expr, error) {
	span := b.span(n.Pos())

	if selector, ok := ast.Unparen(n.Fun).(*ast.SelectorExpr); ok {
		if selection, hasSelection := b.info.Selections[selector]; hasSelection && selection.Kind() == types.MethodVal {
			return b.buildMethodCall(n, selector, selection)
		}
	}

	if function, nameIdent := b.staticCallee(n.Fun); function != nil {
		if function.Pkg() == nil || !b.unit.Owns(function.Pkg().Path()) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call outside the translated unit", Span: span}
		}
		signature := function.Type().(*types.Signature)
		if signature.Recv() != nil {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic call", Span: span}
		}
		call := &Call{Pkg: function.Pkg().Path(), Callee: function.Name()}
		if signature.TypeParams() != nil {
			// An instantiated generic call: the checker's substituted
			// signature types the arguments and results, and the type
			// arguments spell explicitly in the generated call.
			instance, hasInstance := b.info.Instances[nameIdent]
			if !hasInstance {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic call without instantiation evidence", Span: span}
			}
			instantiated, ok := instance.Type.(*types.Signature)
			if !ok {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic call without instantiation evidence", Span: span}
			}
			signature = instantiated
			for i := range instance.TypeArgs.Len() {
				argType, err := b.typeOf(instance.TypeArgs.At(i), span)
				if err != nil {
					return nil, err
				}
				call.TypeArgs = append(call.TypeArgs, argType)
			}
			b.use("genericCall")
		}
		if err := b.buildCallArgsResults(n, signature, &call.Args, &call.Results); err != nil {
			return nil, err
		}
		b.use("call")
		return call, nil
	}

	// Dynamic: any callee expression of function type.
	fun, err := b.buildExpr(n.Fun)
	if err != nil {
		return nil, err
	}
	if fun.Type().Kind != KindFunc {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call of " + fun.Type().Go, Span: span}
	}
	signature, ok := b.info.Types[ast.Unparen(n.Fun)].Type.Underlying().(*types.Signature)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call without signature evidence", Span: span}
	}
	out := &DynCall{Fun: fun}
	if err := b.buildCallArgsResults(n, signature, &out.Args, &out.Results); err != nil {
		return nil, err
	}
	b.use("dynCall")
	return out, nil
}

// staticCallee resolves a plain or package-qualified identifier naming a
// package-level function, by object identity — never spelling. The name
// identifier keys the checker's instantiation evidence for generics.
func (b *builder) staticCallee(fun ast.Expr) (*types.Func, *ast.Ident) {
	unwrapped := ast.Unparen(fun)
	// Explicit instantiation (F[T] or F[T, U]) names the function under
	// the index expression; a non-function object there (a map or slice
	// index) simply resolves to no static callee.
	switch indexed := unwrapped.(type) {
	case *ast.IndexExpr:
		unwrapped = ast.Unparen(indexed.X)
	case *ast.IndexListExpr:
		unwrapped = ast.Unparen(indexed.X)
	}
	switch f := unwrapped.(type) {
	case *ast.Ident:
		function, _ := b.info.Uses[f].(*types.Func)
		return function, f
	case *ast.SelectorExpr:
		if _, hasSelection := b.info.Selections[f]; hasSelection {
			return nil, nil
		}
		if base, isIdent := ast.Unparen(f.X).(*ast.Ident); isIdent {
			if _, isPkg := b.info.Uses[base].(*types.PkgName); isPkg {
				function, _ := b.info.Uses[f.Sel].(*types.Func)
				return function, f.Sel
			}
		}
	}
	return nil, nil
}

func (b *builder) buildMethodCall(n *ast.CallExpr, selector *ast.SelectorExpr, selection *types.Selection) (Expr, error) {
	span := b.span(n.Pos())
	method := selection.Obj().(*types.Func)
	if method.Pkg() == nil || !b.unit.Owns(method.Pkg().Path()) {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call outside the translated unit", Span: span}
	}
	recv, err := b.buildExpr(selector.X)
	if err != nil {
		return nil, err
	}
	if recv.Type().Kind == KindIface {
		return b.buildIfaceMethodCall(n, recv, method)
	}
	signature := method.Type().(*types.Signature)
	if signature.TypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic method call", Span: span}
	}
	// Receiver flavors are exact through the generated method function:
	// a value receiver clones inside (the caller's instance is never
	// mutated) after any pointer caller dereferences at the call; a
	// pointer receiver takes the possibly-nil pointer itself — Go runs
	// pointer-receiver methods on nil receivers, and only the body's own
	// dereferences panic.
	recvNamed, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	pointerRecv := false
	if pointerType, isPointer := signature.Recv().Type().(*types.Pointer); isPointer {
		pointerRecv = true
		recvNamed, ok = types.Unalias(pointerType.Elem()).(*types.Named)
	}
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method on unnamed receiver type", Span: span}
	}
	// A pointer-receiver method needs an addressable instance; a value
	// receiver copies at the call and admits any reviewed carrier
	// (scalars copy as JS values; struct instances clone inside).
	if pointerRecv && recv.Type().Kind != KindPointer && recv.Type().Kind != KindStruct {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "pointer-receiver method call on " + recv.Type().Go, Span: span}
	}

	out := &MethodCall{
		Pkg:         method.Pkg().Path(),
		TypeName:    recvNamed.Obj().Name(),
		PointerRecv: pointerRecv,
		Recv:        recv,
		Method:      method.Name(),
	}
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
// integer-to-integer of any widths, integer-to-float64, and same-carrier
// conversions between named types (identity on the shared carrier).
func (b *builder) checkConversion(from, to Type, span Span) error {
	if from.Kind.Integer() && to.Kind.Integer() {
		return nil
	}
	if from.Kind.Integer() && to.Kind == KindFloat64 {
		return nil
	}
	if from.Kind == to.Kind {
		switch from.Kind {
		case KindString, KindBool, KindFloat64:
			return nil
		}
	}
	return &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION",
		Construct: "conversion from " + from.Go + " to " + to.Go, Span: span}
}
