// Unary-operation lowering: address-of over identity carriers, boxed
// cells, and fresh allocations (&T{...} of structs, arrays, externals,
// and cell-carried pointees); negation and complement.
package ir

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *builder) buildUnary(n *ast.UnaryExpr, resultType types.Type) (Expr, error) {
	span := b.span(n.Pos())
	switch n.Op {
	case token.AND:
		// Heap allocation of a struct literal: &T{...}.
		if lit, ok := ast.Unparen(n.X).(*ast.CompositeLit); ok {
			return b.buildStructNew(lit, resultType)
		}
		// &x on a boxed local: the pointer is the cell itself.
		if ident, isIdent := ast.Unparen(n.X).(*ast.Ident); isIdent {
			if variable, isBoxed := b.boxedVar(ident); isBoxed {
				elemType, err := b.typeOf(variable.Type(), span)
				if err != nil {
					return nil, err
				}
				if boxable(elemType.Kind) {
					t, err := b.typeOf(resultType, span)
					if err != nil {
						return nil, err
					}
					b.use("boxedRef")
					return &BoxedRef{Cell: cellName(ident.Name), T: t}, nil
				}
			}
		}
		// &x on an addressable struct, array, or external value: the
		// pointer is the very instance — whole-value stores overwrite in
		// place, so the alias observes them exactly like Go's memory
		// write.
		x, err := b.buildExpr(ast.Unparen(n.X))
		if err != nil {
			return nil, err
		}
		if x.Type().Kind != KindStruct && x.Type().Kind != KindExternal && x.Type().Kind != KindArray {
			// &s.f where the field is not an identity carrier (a map, slice,
			// scalar, pointer, interface, or function): the field is stored
			// as one stable per-instance cell (the whole-unit scan proved
			// this address is taken), and the pointer IS that cell — so
			// repeated &s.f is the same pointer and a store through it
			// mutates the field.
			if field, isField := x.(*FieldLoad); isField && field.Cell {
				t, err := b.typeOf(resultType, span)
				if err != nil {
					return nil, err
				}
				b.use("fieldCellRef")
				return &FieldCellRef{Base: field.X, Field: field.Field, T: t}, nil
			}
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "address of " + x.Type().Go, Span: span}
		}
		switch x.(type) {
		case *VarRef, *FieldLoad, *SliceGet, *Deref, *ArrayGet:
			t, err := b.typeOf(resultType, span)
			if err != nil {
				return nil, err
			}
			b.use("addrOf")
			return &AddrOf{X: x, T: t}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "address of a non-addressable expression", Span: span}
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
	if t.Elem != nil && t.Elem.Kind == KindExternal {
		if len(lit.Elts) != 0 {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "composite literal of " + t.Elem.Go, Span: span}
		}
		// &T{} of an external type: a fresh zero handle is the pointer.
		b.use("externZero")
		return &AddrOf{X: &ExternZero{T: *t.Elem}, T: t}, nil
	}
	if t.Elem != nil && t.Elem.Kind == KindArray {
		// &[N]T{...}: the fresh array carrier is the pointer.
		built, err := b.buildExprAs(lit, *t.Elem)
		if err != nil {
			return nil, err
		}
		b.use("addrOf")
		return &AddrOf{X: built, T: t}, nil
	}
	if t.Elem != nil && boxable(t.Elem.Kind) {
		// &T{...} of a cell-carried pointee: a fresh cell holding the
		// literal value.
		built, err := b.buildExprAs(lit, *t.Elem)
		if err != nil {
			return nil, err
		}
		b.use("new:cell")
		return &CellNew{Zero: built, T: t}, nil
	}
	return b.buildStructLit(lit, t)
}
