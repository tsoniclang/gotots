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

// ExternLit constructs an external struct value through its reviewed
// keyed-literal constructor stub (one obligation per distinct field
// set): a typed fail-closed contract the emulation layer implements.
type ExternLit struct {
	T      Type
	Symbol string
	Values []Expr
}

func (*ExternLit) expr()        {}
func (e *ExternLit) Type() Type { return e.T }

// ExternFieldRead reads one exported field of an external struct value
// through its typed stub (fail-closed until assembly).
type ExternFieldRead struct {
	X      Expr
	Symbol string
	Pkg    string
	T      Type
}

func (*ExternFieldRead) expr()        {}
func (e *ExternFieldRead) Type() Type { return e.T }

// ExternEqual compares two external struct values through the type's
// typed $eq$ stub (Go's field-wise value equality, emulation-owned).
type ExternEqual struct {
	L, R     Expr
	Pkg      string
	TypeName string
	Negate   bool
}

func (*ExternEqual) expr()        {}
func (e *ExternEqual) Type() Type { return Type{Kind: KindBool, Go: "bool"} }

// ExternToOwned converts an external struct value to the OWNED named
// struct sharing its underlying: the class constructs from per-field
// read stubs (declaration order).
type ExternToOwned struct {
	X            Expr
	To           Type
	FieldSymbols []string
}

func (*ExternToOwned) expr()        {}
func (e *ExternToOwned) Type() Type { return e.To }

// ExternalMethodCall dispatches a method on an external receiver (the
// handle itself, whatever its carrier kind) by canonical identity.
type ExternalMethodCall struct {
	Pkg      string
	TypeName string
	Method   string
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
		return nil, &Unsupported{Kind: KindGenericExternalMethodCall, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "generic external method call", Span: span}
	}
	recvNamed, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	if pointerType, isPointer := signature.Recv().Type().(*types.Pointer); isPointer {
		recvNamed, ok = types.Unalias(pointerType.Elem()).(*types.Named)
	}
	if !ok || recvNamed.Obj().Pkg() == nil {
		return nil, &Unsupported{Kind: KindMethodCallOutsideTheTranslatedUnitUnqualified, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method call outside the translated unit (unqualified)", Span: span}
	}
	_, pointerRecv := signature.Recv().Type().(*types.Pointer)
	if (signature.TypeParams() != nil && signature.TypeParams().Len() > 0) ||
		(signature.RecvTypeParams() != nil && signature.RecvTypeParams().Len() > 0) ||
		SignatureMentionsTypeParam(signature) {
		return nil, &Unsupported{Kind: KindCallToAGenericExternalMethod, Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
			Construct: "call to a generic external method (" + method.Name() + ")", Span: span}
	}
	if err := b.unit.AddExternalMethod(recvNamed, method); err != nil {
		return nil, err
	}
	out := &ExternalMethodCall{
		Pkg:         recvNamed.Obj().Pkg().Path(),
		TypeName:    recvNamed.Obj().Name(),
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
