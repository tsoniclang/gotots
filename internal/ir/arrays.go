// Fixed-array semantics: [N]T is a value type. Copies happen at Go copy
// boundaries, whole-value stores overwrite elements in place (so slices
// over the array and element aliases observe them, exactly like Go's
// memory write), bounds panic with Go's exact message, and slicing an
// addressable array yields a slice sharing its storage.
package ir

import (
	"go/ast"
	"go/token"
)

// ArrayLit is a fixed-array composite literal: positional values with
// explicit zeros for the unfilled tail.
type ArrayLit struct {
	T      Type
	Values []Expr
}

// ArrayGet loads one element with Go's exact bounds panic.
type ArrayGet struct {
	X     Expr
	Index Expr
	T     Type // element type
}

// ArrayTarget assigns one element of an addressable array; struct and
// nested-array elements overwrite in place.
type ArrayTarget struct {
	X     Expr
	Index Expr
}

// ArraySliceView is arr[low:high] over an addressable array: a slice
// sharing the array's storage (capacity runs to the array's end).
type ArraySliceView struct {
	X    Expr
	Low  Expr // nil means 0
	High Expr // nil means len
	T    Type // the resulting slice type
}

// ArrayLenExpr is len(arr) for a non-constant operand: the operand
// evaluates and the fixed length results.
type ArrayLenExpr struct{ X Expr }

// ArrayEqual compares two arrays element-wise in index order with the
// element carrier's exact equality.
type ArrayEqual struct {
	L, R   Expr
	Negate bool
}

func (*ArrayLit) expr()       {}
func (*ArrayGet) expr()       {}
func (*ArraySliceView) expr() {}
func (*ArrayLenExpr) expr()   {}
func (*ArrayEqual) expr()     {}
func (*ArrayTarget) target()  {}

func (a *ArrayLit) Type() Type       { return a.T }
func (a *ArrayGet) Type() Type       { return a.T }
func (a *ArraySliceView) Type() Type { return a.T }
func (a *ArrayLenExpr) Type() Type   { return intType }
func (a *ArrayEqual) Type() Type     { return boolType }

// buildArrayLit lowers [N]T{...}: positional values bind element copies;
// the unfilled tail is explicit zeros. Keyed indexes are a separate
// class.
func (b *builder) buildArrayLit(lit *ast.CompositeLit, t Type) (Expr, error) {
	span := b.span(lit.Pos())
	out := &ArrayLit{T: t}
	for _, element := range lit.Elts {
		if _, isKeyed := element.(*ast.KeyValueExpr); isKeyed {
			return nil, &Unsupported{Kind: KindKeyedArrayLiteral, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "keyed array literal", Span: span}
		}
		value, err := b.buildExprAs(element, *t.Elem)
		if err != nil {
			return nil, err
		}
		out.Values = append(out.Values, value)
	}
	for int64(len(out.Values)) < t.ArrayLen {
		zero, err := zeroValue(*t.Elem, span)
		if err != nil {
			return nil, err
		}
		out.Values = append(out.Values, zero)
	}
	b.use("arrayLiteral")
	return out, nil
}

// buildArrayEqual compares arrays whose element carriers have exact
// direct equality (scalars, strings, booleans, pointers).
func (b *builder) buildArrayEqual(left, right Expr, op token.Token, span Span) (Expr, error) {
	elem := left.Type().Elem
	if elem == nil || !(elem.Kind.Integer() || elem.Kind.Float() ||
		elem.Kind == KindString || elem.Kind == KindBool || elem.Kind == KindPointer) {
		elemGo := left.Type().Go
		if elem != nil {
			elemGo = elem.Go
		}
		return nil, &Unsupported{Kind: KindEqualityOnArrayOf, Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "equality on array of " + elemGo, Span: span}
	}
	b.use("arrayEqual")
	return &ArrayEqual{L: left, R: right, Negate: op == token.NEQ}, nil
}
