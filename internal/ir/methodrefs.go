// Method references as first-class function values. A method value x.M
// captures its receiver at evaluation (copying value receivers, exactly
// Go); a method expression T.M is the generated receiver-first method
// function itself, so it needs no wrapper at all.
package ir

import (
	"go/ast"
	"go/types"
)

// MethodValue is x.M used as a function value.
type MethodValue struct {
	Pkg      string
	TypeName string // empty for an interface receiver
	Method   string
	// PointerRecv reports the method's receiver flavor.
	PointerRecv bool
	// NilCheckRecv dereferences a pointer caller of a value-receiver
	// method at evaluation — Go panics there, not at the call.
	NilCheckRecv bool
	// Iface dispatches through the receiver union's closed member set at
	// call time; a nil interface panics at evaluation.
	Iface bool
	// Results carries the method's result types for dynamic-dispatch
	// casting when Iface is set.
	Results []Type
	Recv    Expr
	T       Type // the KindFunc type of the method value
}

func (*MethodValue) expr()        {}
func (m *MethodValue) Type() Type { return m.T }

// buildMethodValue lowers x.M (selection kind MethodVal).
func (b *builder) buildMethodValue(n *ast.SelectorExpr, selection *types.Selection, valueType types.Type) (Expr, error) {
	span := b.span(n.Pos())
	method := selection.Obj().(*types.Func)
	if method.Pkg() == nil || !b.unit.Owns(method.Pkg().Path()) {
		callee := "unqualified"
		if method.Pkg() != nil {
			callee = method.Pkg().Path() + "." + method.Name()
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method value outside the translated unit (" + callee + ")", Span: span}
	}
	signature := method.Type().(*types.Signature)
	if signature.TypeParams() != nil || signature.RecvTypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic method value", Span: span}
	}
	recv, err := b.buildExpr(n.X)
	if err != nil {
		return nil, err
	}
	recv, err = b.chainPromotedReceiver(recv, b.info.Types[n.X].Type, selection, span)
	if err != nil {
		return nil, err
	}
	t, err := b.typeOf(valueType, span)
	if err != nil {
		return nil, err
	}
	out := &MethodValue{Pkg: method.Pkg().Path(), Method: method.Name(), Recv: recv, T: t}

	if recv.Type().Kind == KindIface {
		out.Iface = true
		results := signature.Results()
		for i := range results.Len() {
			result, err := b.typeOf(results.At(i).Type(), span)
			if err != nil {
				return nil, err
			}
			out.Results = append(out.Results, result)
		}
		b.use("ifaceMethodValue")
		return out, nil
	}

	recvNamed, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	if pointerType, isPointer := signature.Recv().Type().(*types.Pointer); isPointer {
		out.PointerRecv = true
		recvNamed, ok = types.Unalias(pointerType.Elem()).(*types.Named)
	}
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method value on an unnamed receiver type", Span: span}
	}
	out.TypeName = recvNamed.Obj().Name()
	if out.PointerRecv && recv.Type().Kind != KindPointer && recv.Type().Kind != KindStruct {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "pointer-receiver method value on " + recv.Type().Go, Span: span}
	}
	// A value-receiver method value made through a pointer dereferences
	// (and so nil-panics) at evaluation, then captures the copy.
	if !out.PointerRecv && recv.Type().Kind == KindPointer {
		out.NilCheckRecv = true
	}
	b.use("methodValue")
	return out, nil
}

// buildMethodExpr lowers T.M / (*T).M (selection kind MethodExpr): the
// generated method function already takes its receiver first and copies
// value receivers inside, so the reference is the bare symbol.
func (b *builder) buildMethodExpr(n *ast.SelectorExpr, selection *types.Selection, valueType types.Type) (Expr, error) {
	span := b.span(n.Pos())
	method := selection.Obj().(*types.Func)
	if method.Pkg() == nil || !b.unit.Owns(method.Pkg().Path()) {
		callee := "unqualified"
		if method.Pkg() != nil {
			callee = method.Pkg().Path() + "." + method.Name()
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method expression outside the translated unit (" + callee + ")", Span: span}
	}
	signature := method.Type().(*types.Signature)
	if signature.TypeParams() != nil || signature.RecvTypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic method expression", Span: span}
	}
	if _, isIface := signature.Recv().Type().Underlying().(*types.Interface); isIface {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "interface method expression", Span: span}
	}
	recvNamed, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	if pointerType, isPointer := signature.Recv().Type().(*types.Pointer); isPointer {
		recvNamed, ok = types.Unalias(pointerType.Elem()).(*types.Named)
	}
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method expression on an unnamed receiver type", Span: span}
	}
	t, err := b.typeOf(valueType, span)
	if err != nil {
		return nil, err
	}
	// (*T).ValueMethod: the method expression's first parameter is *T,
	// but the generated value-receiver function takes T. An adapter
	// dereferences (a nil pointer panics) and the value copies on entry.
	_, exprRecvPointer := signature2FirstParam(valueType).(*types.Pointer)
	_, methodRecvPointer := signature.Recv().Type().(*types.Pointer)
	if exprRecvPointer && !methodRecvPointer {
		params := make([]Var, 0, signature.Params().Len())
		for i := range signature.Params().Len() {
			pt, err := b.typeOf(signature.Params().At(i).Type(), span)
			if err != nil {
				return nil, err
			}
			params = append(params, Var{Name: signature.Params().At(i).Name(), Type: pt})
		}
		results := make([]Type, 0, signature.Results().Len())
		for i := range signature.Results().Len() {
			rt, err := b.typeOf(signature.Results().At(i).Type(), span)
			if err != nil {
				return nil, err
			}
			results = append(results, rt)
		}
		recvT, err := b.typeOf(signature.Recv().Type(), span)
		if err != nil {
			return nil, err
		}
		b.use("methodExpr:derefAdapter")
		return &MethodExprAdapter{
			Pkg: method.Pkg().Path(), Method: recvNamed.Obj().Name() + "$" + method.Name(),
			RecvValue: recvT, Params: params, Results: results, T: t,
		}, nil
	}
	b.use("methodExpr")
	return &FuncRef{Pkg: method.Pkg().Path(), Name: recvNamed.Obj().Name() + "$" + method.Name(), T: t}, nil
}

// signature2FirstParam returns the first parameter type of a function
// type (a method expression's spelled receiver parameter), or nil.
func signature2FirstParam(t types.Type) types.Type {
	sig, ok := t.Underlying().(*types.Signature)
	if !ok || sig.Params().Len() == 0 {
		return nil
	}
	return sig.Params().At(0).Type()
}

// MethodExprAdapter is (*T).ValueMethod: an arrow taking *T,
// dereferencing (nil panics) and forwarding to the value-receiver
// method, which copies on entry.
type MethodExprAdapter struct {
	Pkg       string
	Method    string
	RecvValue Type // the value receiver type T
	Params    []Var
	Results   []Type
	T         Type
}

func (*MethodExprAdapter) expr()        {}
func (a *MethodExprAdapter) Type() Type { return a.T }
