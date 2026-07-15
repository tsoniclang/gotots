package ir

import (
	"go/ast"
	"go/token"
	"go/types"
)

// buildBinary lowers one binary operation with exact operand semantics,
// including nil comparisons for pointers and maps.
func (b *builder) buildBinary(n *ast.BinaryExpr, resultType types.Type) (Expr, error) {
	span := b.span(n.Pos())

	// Nil comparisons: one operand is the untyped nil.
	if n.Op == token.EQL || n.Op == token.NEQ {
		leftNil := b.info.Types[n.X].IsNil()
		rightNil := b.info.Types[n.Y].IsNil()
		if leftNil || rightNil {
			operandAST := n.X
			if leftNil {
				operandAST = n.Y
			}
			operand, err := b.buildExpr(operandAST)
			if err != nil {
				return nil, err
			}
			if !operand.Type().Kind.Nilable() {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "nil comparison on " + operand.Type().Go, Span: span}
			}
			b.use("isNil")
			return &IsNil{X: operand, Negate: n.Op == token.NEQ}, nil
		}
	}

	left, err := b.buildExpr(n.X)
	if err != nil {
		return nil, err
	}
	right, err := b.buildExpr(n.Y)
	if err != nil {
		return nil, err
	}
	t, err := b.typeOf(resultType, span)
	if err != nil {
		return nil, err
	}
	operand := left.Type()

	switch n.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
		token.AND, token.OR, token.XOR, token.AND_NOT,
		token.SHL, token.SHR:
		if n.Op == token.ADD && operand.Kind == KindString {
			// String concatenation is exact under direct JS +.
		} else if !operand.Kind.Integer() && !operand.Kind.Float() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + n.Op.String() + " on " + operand.Go, Span: span}
		}
		if operand.Kind.Float() && (n.Op == token.REM || n.Op == token.AND || n.Op == token.OR ||
			n.Op == token.XOR || n.Op == token.AND_NOT || n.Op == token.SHL || n.Op == token.SHR) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + n.Op.String() + " on " + operand.Go, Span: span}
		}
		if operand.Kind == KindFloat32 {
			// Exact float32 rounding points are not lowered yet.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "float32 arithmetic", Span: span}
		}
	case token.EQL, token.NEQ:
		// Equality is exact for scalar carriers (UTF-8 string equality
		// coincides with JS code-unit equality) and for pointer identity;
		// arrays compare element-wise in index order.
		if operand.Kind == KindArray {
			return b.buildArrayEqual(left, right, n.Op, span)
		}
		if operand.Kind == KindMap || operand.Kind == KindStruct || operand.Kind == KindExternal {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "equality on " + operand.Go, Span: span}
		}
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		// Byte-string carriers compare code units, which ARE the Go
		// bytes, so JS ordering coincides with Go's byte-wise ordering.
		if operand.Kind != KindString && !operand.Kind.Integer() && !operand.Kind.Float() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "ordering on " + operand.Go, Span: span}
		}
	case token.LAND, token.LOR:
		// Short-circuit boolean semantics match JS exactly.
	default:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + n.Op.String(), Span: span}
	}
	b.use("binary:" + n.Op.String())
	return &Binary{Op: n.Op, L: left, R: right, T: t}, nil
}
