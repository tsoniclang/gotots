// Byte-string semantics: a Go string is an immutable byte sequence,
// carried as a JS string whose every code unit is one byte (0x00-0xFF).
// The carrier is canonical — each byte sequence has exactly one
// representation — so equality, concatenation, and byte-wise ordering
// are direct, and indexing, byte slicing, and rune-decoding range are
// exact for arbitrary bytes, valid UTF-8 or not.
package ir

import (
	"fmt"
	"go/ast"
	"go/types"
)

// StringIndex loads one byte (uint8) with Go's exact bounds panic.
type StringIndex struct {
	X     Expr
	Index Expr
}

// StringSlice is s[low:high] by byte offsets with Go's exact
// slice-bounds panics; mid-rune cuts are exact byte results.
type StringSlice struct {
	X    Expr
	Low  Expr // nil means 0
	High Expr // nil means len
	T    Type
}

// RangeString iterates a string's runes: the index is the starting byte
// offset, the value is the decoded rune (int32), and invalid UTF-8
// yields U+FFFD advancing one byte — exactly Go's range.
type RangeString struct {
	Index string // "" when omitted
	Value string // "" when omitted
	X     Expr
	Body  *Block
}

func (*StringIndex) expr() {}
func (*StringSlice) expr() {}
func (*RangeString) stmt() {}

func (s *StringIndex) Type() Type { return Type{Kind: KindUint8, Go: "byte"} }
func (s *StringSlice) Type() Type { return s.T }

// buildStringIndex lowers s[i] over the byte carrier.
func (b *builder) buildStringIndex(operand Expr, index ast.Expr) (Expr, error) {
	built, err := b.buildExpr(index)
	if err != nil {
		return nil, err
	}
	b.use("index:string")
	return &StringIndex{X: operand, Index: built}, nil
}

// buildRangeString lowers range over a string: the operand evaluates
// once (strings are immutable, so no snapshot is needed).
func (b *builder) buildRangeString(n *ast.RangeStmt, operand Expr) (Stmt, error) {
	span := b.span(n.Pos())
	out := &RangeString{X: operand}
	name := func(e ast.Expr) (string, error) {
		if e == nil {
			return "", nil
		}
		ident, ok := e.(*ast.Ident)
		if !ok {
			return "", &Unsupported{Kind: KindRangeVariableIsNotAnIdentifier, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range variable is not an identifier", Span: span}
		}
		if ident.Name == "_" {
			return "", nil
		}
		return b.bindNameOf(ident), nil
	}
	var err error
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
	b.use("range:string")
	return out, nil
}

// byteStringLiteral spells a Go string constant as a JS literal over the
// byte carrier: one code unit per byte, printable ASCII kept readable,
// every other byte as a \xNN escape.
func byteStringLiteral(text string) string {
	var out []byte
	out = append(out, '"')
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c == '"' || c == '\\':
			out = append(out, '\\', c)
		case c >= 0x20 && c <= 0x7e:
			out = append(out, c)
		default:
			out = append(out, []byte(fmt.Sprintf("\\x%02x", c))...)
		}
	}
	return string(append(out, '"'))
}

// buildStringSlice lowers s[low:high] over the byte carrier.
func (b *builder) buildStringSlice(operand Expr, n *ast.SliceExpr) (Expr, error) {
	out := &StringSlice{X: operand, T: operand.Type()}
	var err error
	if n.Low != nil {
		if out.Low, err = b.buildExpr(n.Low); err != nil {
			return nil, err
		}
	}
	if n.High != nil {
		if out.High, err = b.buildExpr(n.High); err != nil {
			return nil, err
		}
	}
	b.use("reslice:string")
	return out, nil
}

// StringConvert is a conversion in the string family, exact over the
// byte carrier: rune/integer to string (UTF-8 encoding, U+FFFD for
// invalid code points), []byte/string round trips (fresh copies), and
// []rune/string round trips (decode with U+FFFD per invalid byte).
type StringConvert struct {
	Op string // fromRune | fromBytes | toBytes | fromRunes | toRunes
	X  Expr
	T  Type
}

func (*StringConvert) expr()        {}
func (s *StringConvert) Type() Type { return s.T }

// buildStringConversion intercepts conversions into and out of the
// string family; ok reports whether the conversion was one.
func (b *builder) buildStringConversion(x Expr, to Type) (Expr, bool) {
	from := x.Type()
	elemKind := func(t Type) Kind {
		if t.Kind != KindSlice || t.Elem == nil {
			return KindInvalid
		}
		return t.Elem.Kind
	}
	switch {
	case from.Kind.Integer() && to.Kind == KindString:
		b.use("convert:runeToString")
		return &StringConvert{Op: "fromRune", X: x, T: to}, true
	case elemKind(from) == KindUint8 && to.Kind == KindString:
		b.use("convert:bytesToString")
		return &StringConvert{Op: "fromBytes", X: x, T: to}, true
	case from.Kind == KindString && elemKind(to) == KindUint8:
		b.use("convert:stringToBytes")
		return &StringConvert{Op: "toBytes", X: x, T: to}, true
	case elemKind(from) == KindInt32 && to.Kind == KindString:
		b.use("convert:runesToString")
		return &StringConvert{Op: "fromRunes", X: x, T: to}, true
	case from.Kind == KindString && elemKind(to) == KindInt32:
		b.use("convert:stringToRunes")
		return &StringConvert{Op: "toRunes", X: x, T: to}, true
	}
	return nil, false
}

// buildUnsafeStringExact admits the ONE reviewed unsafe.String form:
// unsafe.String(&b[0], len(b)) with both operands reading the SAME
// byte-slice binding. Go specifies the result as the string of those
// len(b) bytes — exactly string(b); the unsafe form only avoids the
// copy, which our string carrier performs regardless. Any other
// unsafe.String shape fails closed.
func (b *builder) buildUnsafeStringExact(n *ast.CallExpr) (Expr, bool, error) {
	span := b.span(n.Pos())
	sel, isSel := ast.Unparen(n.Fun).(*ast.SelectorExpr)
	if !isSel {
		return nil, false, nil
	}
	builtin, isBuiltin := b.info.Uses[sel.Sel].(*types.Builtin)
	if !isBuiltin || builtin.Name() != "String" {
		return nil, false, nil
	}
	reject := func() (Expr, bool, error) {
		return nil, true, &Unsupported{Kind: KindNonFieldSelector, Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
			Construct: "unsafe.String outside the reviewed exact form (&b[0], len(b))", Span: span}
	}
	if len(n.Args) != 2 {
		return reject()
	}
	sliceExpr, indexExpr, isAddr := b.addressedSliceElem(n.Args[0])
	if !isAddr {
		return reject()
	}
	sliceIdent, isIdent := ast.Unparen(sliceExpr).(*ast.Ident)
	if !isIdent {
		return reject()
	}
	if indexValue, ok := b.info.Types[indexExpr]; !ok || indexValue.Value == nil || indexValue.Value.String() != "0" {
		return reject()
	}
	lenCall, isCall := ast.Unparen(n.Args[1]).(*ast.CallExpr)
	if !isCall || len(lenCall.Args) != 1 {
		return reject()
	}
	if lenBuiltin, ok := b.info.Uses[identOf(lenCall.Fun)].(*types.Builtin); !ok || lenBuiltin.Name() != "len" {
		return reject()
	}
	lenIdent, isLenIdent := ast.Unparen(lenCall.Args[0]).(*ast.Ident)
	if !isLenIdent || b.info.ObjectOf(lenIdent) == nil ||
		b.info.ObjectOf(lenIdent) != b.info.ObjectOf(sliceIdent) {
		return reject()
	}
	slice, err := b.buildExpr(sliceExpr)
	if err != nil {
		return nil, true, err
	}
	elem := slice.Type().Elem
	if elem == nil || elem.Kind != KindUint8 {
		return reject()
	}
	b.use("unsafe:stringFromBytes")
	return &StringConvert{Op: "fromBytes", X: slice, T: Type{Kind: KindString, Go: "string"}}, true, nil
}

// identOf unwraps an identifier expression (nil when it is not one).
func identOf(e ast.Expr) *ast.Ident {
	ident, _ := ast.Unparen(e).(*ast.Ident)
	return ident
}
