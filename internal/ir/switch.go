package ir

import (
	"go/ast"
	"go/token"
)

// buildSwitch lowers a Go expression switch. The reviewed carriers make
// JS switch semantics coincide exactly: the tag is evaluated once, case
// expressions are evaluated in source order and compared with the
// carrier's identity equality, the first match wins, and the default
// clause runs only when nothing matches regardless of its position.
// Fallthrough transfers into the next clause's body on both sides.
func (b *builder) buildSwitch(n *ast.SwitchStmt) (Stmt, error) {
	span := b.span(n.Pos())
	out := &SwitchStmt{}
	if n.Init != nil {
		init, err := b.buildStmt(n.Init)
		if err != nil {
			return nil, err
		}
		out.Init = init
	}

	if n.Tag == nil {
		out.Tag = &Const{T: Type{Kind: KindBool, Go: "bool"}, Value: "true"}
	} else {
		tag, err := b.buildExpr(n.Tag)
		if err != nil {
			return nil, err
		}
		out.Tag = tag
	}
	tagType := out.Tag.Type()
	switch {
	case tagType.Kind == KindBool, tagType.Kind == KindString,
		tagType.Kind == KindPointer, tagType.Kind.Integer(), tagType.Kind.Float():
		// Equality on these carriers is exact under JS case matching.
	default:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "switch tag of " + tagType.Go, Span: span}
	}

	for _, clauseStmt := range n.Body.List {
		clause := clauseStmt.(*ast.CaseClause)
		built := SwitchClause{}
		for _, value := range clause.List {
			expr, err := b.buildExprAs(value, tagType)
			if err != nil {
				return nil, err
			}
			if _, isNil := expr.(*NilConst); !isNil && expr.Type().Kind != tagType.Kind {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
					Construct: "switch case of " + expr.Type().Go + " against tag of " + tagType.Go, Span: b.span(value.Pos())}
			}
			built.Values = append(built.Values, expr)
		}

		// A trailing fallthrough (Go permits it only as the clause's last
		// statement) becomes the clause's transfer flag.
		bodyStmts := clause.Body
		if last := len(bodyStmts) - 1; last >= 0 {
			if branch, isBranch := bodyStmts[last].(*ast.BranchStmt); isBranch && branch.Tok == token.FALLTHROUGH {
				built.Fallthrough = true
				bodyStmts = bodyStmts[:last]
			}
		}
		body, err := b.buildBreakableBody(&ast.BlockStmt{List: bodyStmts})
		if err != nil {
			return nil, err
		}
		built.Body = body
		out.Clauses = append(out.Clauses, built)
	}
	b.use("switch")
	return out, nil
}
