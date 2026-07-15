// External-value semantics: a named type outside the translated unit is
// an opaque handle. Zero construction, value copies at Go copy
// boundaries, whole-value stores, and method calls are all reviewed
// external contracts dispatched by canonical identity — runtime
// fail-closed (GOTOTS_EXTERNAL_UNIMPLEMENTED) until the emulation layer
// registers behavior. Value-receiver method contracts must not mutate
// their receiver: Go runs them on copies.
package ir

import (
	"go/ast"
	"go/types"
)

// ExternZero constructs the zero value of an external type through its
// zero contract.
type ExternZero struct{ T Type }

func (*ExternZero) expr()        {}
func (e *ExternZero) Type() Type { return e.T }

// ExternalMethodCall dispatches a method on an external receiver (the
// handle itself, whatever its carrier kind) by canonical identity.
type ExternalMethodCall struct {
	TypeID string // "<package path>.<TypeName>"
	Method string
	// PointerRecv reports the declared receiver flavor: a value receiver
	// captures a copy when deferred, exactly like owned methods.
	PointerRecv bool
	Recv        Expr
	Args        []Expr
	Results     []Type
}

func (*ExternalMethodCall) expr() {}
func (m *ExternalMethodCall) Type() Type {
	if len(m.Results) == 1 {
		return m.Results[0]
	}
	return Type{Kind: KindInvalid, Go: "()"}
}

// ExternVar reads an external package variable through its contract by
// canonical identity; the emulation must return an identity-stable
// value (sentinel comparisons rely on it).
type ExternVar struct {
	ID string
	T  Type
}

func (*ExternVar) expr()        {}
func (e *ExternVar) Type() Type { return e.T }

// buildExternalMethodCall admits a method call on a receiver declared
// outside the unit when the whole signature resolves in the reviewed
// type set.
func (b *builder) buildExternalMethodCall(n *ast.CallExpr, selector *ast.SelectorExpr, method *types.Func, recv Expr) (Expr, error) {
	span := b.span(n.Pos())
	signature := method.Type().(*types.Signature)
	if signature.TypeParams() != nil || signature.RecvTypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic external method call", Span: span}
	}
	recvNamed, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	if pointerType, isPointer := signature.Recv().Type().(*types.Pointer); isPointer {
		recvNamed, ok = types.Unalias(pointerType.Elem()).(*types.Named)
	}
	if !ok || recvNamed.Obj().Pkg() == nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call outside the translated unit (unqualified)", Span: span}
	}
	_, pointerRecv := signature.Recv().Type().(*types.Pointer)
	out := &ExternalMethodCall{
		TypeID:      recvNamed.Obj().Pkg().Path() + "." + recvNamed.Obj().Name(),
		Method:      method.Name(),
		PointerRecv: pointerRecv,
		Recv:        recv,
	}
	if err := b.buildCallArgsResults(n, signature, &out.Args, &out.Results); err != nil {
		return nil, err
	}
	b.use("externMethodCall")
	return out, nil
}
