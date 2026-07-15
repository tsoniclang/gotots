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
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "range variable is not an identifier", Span: span}
		}
		if ident.Name == "_" {
			return "", nil
		}
		return ident.Name, nil
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
