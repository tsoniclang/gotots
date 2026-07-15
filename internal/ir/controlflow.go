// Control-flow and declaration statement lowering: local declarations,
// inc/dec, if, for, and returns.
package ir

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (b *builder) buildDeclStmt(n *ast.DeclStmt) (Stmt, error) {
	span := b.span(n.Pos())
	decl, ok := n.Decl.(*ast.GenDecl)
	if ok && decl.Tok == token.CONST {
		// Local constants are compile-time values: every use folds to its
		// exact constant at the use site, so the declaration emits nothing.
		b.use("localConst")
		return &Block{}, nil
	}
	if !ok || decl.Tok != token.VAR {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "non-var declaration statement", Span: span}
	}
	out := &DeclStmt{}
	for _, spec := range decl.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "non-value var spec", Span: span}
		}
		for i, name := range value.Names {
			if name.Name == "_" {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "blank variable declaration", Span: span}
			}
			object := b.info.Defs[name]
			if object == nil {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "var without typed definition", Span: span}
			}
			t, err := b.typeOf(object.Type(), span)
			if err != nil {
				return nil, err
			}
			out.Names = append(out.Names, name.Name)
			out.Types = append(out.Types, t)
			if i < len(value.Values) {
				built, err := b.buildExprAs(value.Values[i], t)
				if err != nil {
					return nil, err
				}
				out.Values = append(out.Values, built)
			} else {
				zero, err := zeroValue(t, span)
				if err != nil {
					return nil, err
				}
				out.Values = append(out.Values, zero)
			}
		}
		if len(value.Values) != 0 && len(value.Values) != len(value.Names) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "multi-value var initializer", Span: span}
		}
	}
	b.use("varDecl")
	return out, nil
}

// buildIncDec lowers x++ / x-- through the shared single-evaluation
// compound-target resolver.
func (b *builder) buildIncDec(n *ast.IncDecStmt) (Stmt, error) {
	span := b.span(n.Pos())
	target, x, err := b.compoundTarget(n.X)
	if err != nil {
		return nil, err
	}
	if !x.Type().Kind.Integer() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "inc/dec of " + x.Type().Go, Span: span}
	}
	op := token.ADD
	if n.Tok == token.DEC {
		op = token.SUB
	}
	one := &Const{T: x.Type(), Value: "1"}
	b.use("incDec")
	return &AssignStmt{
		Targets: []Target{target},
		Values:  []Expr{&Binary{Op: op, L: x, R: one, T: x.Type()}},
	}, nil
}

func (b *builder) buildIf(n *ast.IfStmt) (Stmt, error) {
	out := &IfStmt{}
	if n.Init != nil {
		init, err := b.buildStmt(n.Init)
		if err != nil {
			return nil, err
		}
		out.Init = init
	}
	cond, err := b.buildExpr(n.Cond)
	if err != nil {
		return nil, err
	}
	out.Cond = cond
	then, err := b.buildBlock(n.Body)
	if err != nil {
		return nil, err
	}
	out.Then = then
	if n.Else != nil {
		built, err := b.buildStmt(n.Else)
		if err != nil {
			return nil, err
		}
		out.Else = built
	}
	b.use("if")
	return out, nil
}

func (b *builder) buildFor(n *ast.ForStmt) (Stmt, error) {
	out := &ForStmt{}
	if n.Init != nil {
		init, err := b.buildStmt(n.Init)
		if err != nil {
			return nil, err
		}
		if _, isSeq := init.(*StmtSeq); isSeq {
			// A boxed (address-taken) loop-clause variable has no
			// expressible cell declaration inside the clause yet.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "address of a loop-clause variable", Span: b.span(n.Pos())}
		}
		out.Init = init
	}
	if n.Cond != nil {
		cond, err := b.buildExpr(n.Cond)
		if err != nil {
			return nil, err
		}
		out.Cond = cond
	}
	if n.Post != nil {
		post, err := b.buildStmt(n.Post)
		if err != nil {
			return nil, err
		}
		out.Post = post
	}
	body, err := b.buildBreakableBody(n.Body)
	if err != nil {
		return nil, err
	}
	out.Body = body
	b.use("for")
	return out, nil
}

func (b *builder) buildReturn(n *ast.ReturnStmt) (Stmt, error) {
	if len(n.Results) == 0 {
		out := &ReturnStmt{}
		// A bare return in a function with named results returns their
		// current values.
		for _, result := range b.namedResults {
			out.Values = append(out.Values, &VarRef{Name: result.Name, T: result.Type})
		}
		b.use("return")
		return out, nil
	}
	// A single multi-result call forwarded as the complete result list.
	if len(n.Results) == 1 {
		if callAST, ok := ast.Unparen(n.Results[0]).(*ast.CallExpr); ok {
			if _, isConversion := b.conversionTarget(callAST); !isConversion {
				if _, isBuiltin := b.builtinCallee(callAST); !isBuiltin {
					if tuple, ok := b.info.Types[callAST].Type.(*types.Tuple); ok && tuple.Len() > 1 {
						call, err := b.buildAnyCall(callAST)
						if err != nil {
							return nil, err
						}
						b.use("return")
						return &ReturnStmt{CallValue: call}, nil
					}
				}
			}
		}
	}
	out := &ReturnStmt{}
	for i, result := range n.Results {
		if i >= len(b.results) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "return arity mismatch", Span: b.span(n.Pos())}
		}
		built, err := b.buildExprAs(result, b.results[i])
		if err != nil {
			return nil, err
		}
		out.Values = append(out.Values, built)
	}
	b.use("return")
	return out, nil
}
