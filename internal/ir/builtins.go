package ir

import (
	"go/ast"

	"go/types"
)

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
		case KindSlice:
			b.use("len:slice")
			return &SliceLen{X: operand}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "len of " + operand.Type().Go, Span: span}
	case "cap":
		operand, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind == KindSlice {
			b.use("cap:slice")
			return &SliceCap{X: operand}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "cap of " + operand.Type().Go, Span: span}
	case "append":
		operand, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind != KindSlice {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "append to " + operand.Type().Go, Span: span}
		}
		if call.Ellipsis.IsValid() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "append with slice expansion", Span: span}
		}
		out := &SliceAppend{X: operand, T: operand.Type()}
		for _, arg := range call.Args[1:] {
			value, err := b.buildExprAs(arg, *operand.Type().Elem)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		b.use("append")
		return out, nil
	case "make":
		t, err := b.typeOf(resultType, span)
		if err != nil {
			return nil, err
		}
		if t.Kind == KindMap && len(call.Args) == 1 {
			b.use("makeMap")
			return &MapMake{T: t}, nil
		}
		if t.Kind == KindSlice && (len(call.Args) == 2 || len(call.Args) == 3) {
			length, err := b.buildExpr(call.Args[1])
			if err != nil {
				return nil, err
			}
			out := &SliceMake{T: t, Length: length}
			if len(call.Args) == 3 {
				capacity, err := b.buildExpr(call.Args[2])
				if err != nil {
					return nil, err
				}
				out.Capacity = capacity
			}
			b.use("makeSlice")
			return out, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "make of " + t.Go, Span: span}
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "builtin " + builtin.Name(), Span: span}
}
