// Range-statement lowering: the operand class selects the loop shape,
// induction variables stay hidden, and range variables bind
// per-iteration copies exactly as the pinned Go version defines.
package ir

import (
	"go/ast"
	"go/token"
)

// buildRange lowers range over a slice: the operand and its length are
// evaluated once, index is an int, and each element loads per iteration.
func (b *builder) buildRange(n *ast.RangeStmt) (Stmt, error) {
	span := b.span(n.Pos())
	if n.Tok != token.DEFINE && n.Tok != token.ILLEGAL {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range with assignment form", Span: span}
	}
	operand, err := b.buildExpr(n.X)
	if err != nil {
		return nil, err
	}
	if operand.Type().Kind.Integer() {
		return b.buildRangeInt(n, operand)
	}
	if operand.Type().Kind == KindArray {
		// The range expression evaluates once; iterating a snapshot view
		// makes body assignments to the source array unobservable to the
		// loop, exactly like Go's array-value evaluation.
		snapshot := b.bindStructValue(operand)
		elem := operand.Type().Elem
		operand = &ArraySliceView{X: snapshot, T: Type{Kind: KindSlice, Go: "[]" + elem.Go, Elem: elem}}
		b.use("range:array")
	}
	if operand.Type().Kind == KindString {
		return b.buildRangeString(n, operand)
	}
	if operand.Type().Kind == KindMap {
		return b.buildRangeMap(n, operand)
	}
	if operand.Type().Kind != KindSlice {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range over " + operand.Type().Go, Span: span}
	}
	out := &RangeSlice{X: operand, VarT: *operand.Type().Elem}
	name := func(e ast.Expr) (string, error) {
		if e == nil {
			return "", nil
		}
		ident, ok := e.(*ast.Ident)
		if !ok {
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range variable is not an identifier", Span: span}
		}
		if ident.Name == "_" {
			return "", nil
		}
		return ident.Name, nil
	}
	if out.Index, err = name(n.Key); err != nil {
		return nil, err
	}
	if out.Value, err = name(n.Value); err != nil {
		return nil, err
	}
	body, err := b.buildBlock(n.Body)
	if err != nil {
		return nil, err
	}
	out.Body = body
	b.use("range:slice")
	return out, nil
}

// buildRangeInt lowers `for i := range n`: n evaluates once and i
// counts from zero to n-1 in n's own carrier.
func (b *builder) buildRangeInt(n *ast.RangeStmt, operand Expr) (Stmt, error) {
	span := b.span(n.Pos())
	if n.Value != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range over an integer with a second variable", Span: span}
	}
	out := &RangeInt{N: operand}
	if n.Key != nil {
		ident, ok := n.Key.(*ast.Ident)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range variable is not an identifier", Span: span}
		}
		if ident.Name != "_" {
			out.Index = ident.Name
		}
	}
	body, err := b.buildBlock(n.Body)
	if err != nil {
		return nil, err
	}
	out.Body = body
	b.use("range:int")
	return out, nil
}

// RangeMap iterates a map's live entries in the carrier's insertion
// order — one of the iteration orders Go permits (Go's own order is
// unspecified), produced deterministically. Entries deleted before
// being reached are not produced and entries added during iteration may
// be produced, both within Go's allowed behaviors. A nil map iterates
// zero times.
type RangeMap struct {
	Key   string // "" when omitted
	Value string // "" when omitted
	ValT  Type
	X     Expr
	Body  *Block
}

func (*RangeMap) stmt() {}

func (b *builder) buildRangeMap(n *ast.RangeStmt, operand Expr) (Stmt, error) {
	span := b.span(n.Pos())
	out := &RangeMap{X: operand, ValT: *operand.Type().Elem}
	name := func(e ast.Expr) (string, error) {
		if e == nil {
			return "", nil
		}
		ident, ok := e.(*ast.Ident)
		if !ok {
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range variable is not an identifier", Span: span}
		}
		if ident.Name == "_" {
			return "", nil
		}
		return ident.Name, nil
	}
	var err error
	if out.Key, err = name(n.Key); err != nil {
		return nil, err
	}
	if out.Value, err = name(n.Value); err != nil {
		return nil, err
	}
	body, err := b.buildBlock(n.Body)
	if err != nil {
		return nil, err
	}
	out.Body = body
	b.use("range:map")
	return out, nil
}
