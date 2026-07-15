package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// opAssignTokens maps compound assignment tokens to their operator.
var opAssignTokens = map[token.Token]token.Token{
	token.ADD_ASSIGN:     token.ADD,
	token.SUB_ASSIGN:     token.SUB,
	token.MUL_ASSIGN:     token.MUL,
	token.QUO_ASSIGN:     token.QUO,
	token.REM_ASSIGN:     token.REM,
	token.AND_ASSIGN:     token.AND,
	token.OR_ASSIGN:      token.OR,
	token.XOR_ASSIGN:     token.XOR,
	token.SHL_ASSIGN:     token.SHL,
	token.SHR_ASSIGN:     token.SHR,
	token.AND_NOT_ASSIGN: token.AND_NOT,
}

func (b *builder) buildAssign(n *ast.AssignStmt) (Stmt, error) {
	span := b.span(n.Pos())

	if operator, isCompound := opAssignTokens[n.Tok]; isCompound {
		return b.buildCompoundAssign(n, operator)
	}

	targets := make([]*ast.Ident, 0, len(n.Lhs))
	for _, lhs := range n.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("assignment to %T", lhs), Span: span}
		}
		if ident.Name == "_" {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to blank identifier", Span: span}
		}
		targets = append(targets, ident)
	}

	// A single multi-result call spreading into all targets.
	multiCall, err := b.multiResultCall(n.Rhs, len(targets))
	if err != nil {
		return nil, err
	}

	switch n.Tok {
	case token.DEFINE:
		out := &DeclStmt{}
		for _, target := range targets {
			object := b.info.Defs[target]
			definition, isNew := object.(*types.Var)
			if !isNew {
				// Go permits := to reassign existing names alongside one new
				// one; that scoping subtlety has its own reviewed lowering
				// later.
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration reusing an existing variable", Span: span}
			}
			t, err := typeOf(definition.Type(), span)
			if err != nil {
				return nil, err
			}
			out.Names = append(out.Names, target.Name)
			out.Types = append(out.Types, t)
		}
		if multiCall != nil {
			out.CallValues = multiCall
			b.use("declMultiCall")
			return out, nil
		}
		if len(n.Rhs) != len(targets) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration arity mismatch", Span: span}
		}
		for _, rhs := range n.Rhs {
			value, err := b.buildExpr(rhs)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		b.use("shortDecl")
		return out, nil

	case token.ASSIGN:
		out := &AssignStmt{}
		for _, target := range targets {
			if _, isUse := b.info.Uses[target].(*types.Var); !isUse {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-variable " + target.Name, Span: span}
			}
			out.Targets = append(out.Targets, target.Name)
		}
		if multiCall != nil {
			out.CallValues = multiCall
			b.use("assignMultiCall")
			return out, nil
		}
		if len(n.Rhs) != len(targets) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment arity mismatch", Span: span}
		}
		for _, rhs := range n.Rhs {
			value, err := b.buildExpr(rhs)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		b.use("assign")
		return out, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment token " + n.Tok.String(), Span: span}
}

// buildCompoundAssign lowers x op= y into x = x op y. For a plain variable
// target the single evaluation of the target address is trivially
// preserved.
func (b *builder) buildCompoundAssign(n *ast.AssignStmt, operator token.Token) (Stmt, error) {
	span := b.span(n.Pos())
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment arity", Span: span}
	}
	target, ok := n.Lhs[0].(*ast.Ident)
	if !ok || target.Name == "_" {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment to non-variable", Span: span}
	}
	left, err := b.buildExpr(target)
	if err != nil {
		return nil, err
	}
	right, err := b.buildExpr(n.Rhs[0])
	if err != nil {
		return nil, err
	}
	operand := left.Type()
	if operand.Kind == KindString && operator != token.ADD {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + operator.String() + " on string", Span: span}
	}
	if operand.Kind == KindBool {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "compound assignment on bool", Span: span}
	}
	if operand.Kind == KindFloat32 {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "float32 arithmetic", Span: span}
	}
	b.use("compoundAssign:" + operator.String())
	return &AssignStmt{
		Targets: []string{target.Name},
		Values:  []Expr{&Binary{Op: operator, L: left, R: right, T: operand}},
	}, nil
}

// multiResultCall recognizes a single call whose tuple result initializes
// several targets; it returns nil when the shape does not apply.
func (b *builder) multiResultCall(rhs []ast.Expr, targetCount int) (*Call, error) {
	if len(rhs) != 1 || targetCount < 2 {
		return nil, nil
	}
	callAST, ok := ast.Unparen(rhs[0]).(*ast.CallExpr)
	if !ok {
		return nil, nil
	}
	if _, isConversion := b.conversionTarget(callAST); isConversion {
		return nil, nil
	}
	tuple, ok := b.info.Types[callAST].Type.(*types.Tuple)
	if !ok || tuple.Len() != targetCount {
		return nil, nil
	}
	return b.buildCall(callAST)
}
