// Boxed-variable cells: a local variable whose address is taken (and
// whose carrier has no object identity of its own) lives in a mutable
// cell { v: T }. The cell is created at the declaration, every read and
// write goes through it, and &x is the cell itself — so pointer
// identity, aliasing, and nil semantics are exact. Fixed arrays,
// structs, and external handles keep their own identity and never box.
package ir

import (
	"go/ast"
	"go/token"
	"go/types"
)

// BoxedDecl declares one cell-carried variable with its initial value.
type BoxedDecl struct {
	Cell string // the emission name (source name + "$b"; Go cannot spell "$")
	Elem Type
	Init Expr
}

// BoxedLoad reads a cell-carried variable.
type BoxedLoad struct {
	Cell string
	T    Type
}

// BoxedTarget assigns a cell-carried variable.
type BoxedTarget struct {
	Cell string
	T    Type
}

// BoxedRef is &x on a cell-carried variable: the cell itself.
type BoxedRef struct {
	Cell string
	T    Type // the pointer type
}

// CellNew is new(T) for a cell-carried pointee: a fresh zero cell.
type CellNew struct {
	Zero Expr
	T    Type // the pointer type
}

func (*BoxedDecl) stmt()         {}
func (*BoxedLoad) expr()         {}
func (*BoxedRef) expr()          {}
func (*CellNew) expr()           {}
func (BoxedTarget) target()      {}
func (x *BoxedLoad) Type() Type  { return x.T }
func (x *BoxedRef) Type() Type   { return x.T }
func (x *CellNew) Type() Type    { return x.T }
func (x BoxedTarget) Type() Type { return x.T }
func (x *BoxedDecl) ElemT() Type { return x.Elem }

// cellName spells the emission name of a boxed variable's cell.
func cellName(source string) string { return source + "$b" }

// boxable reports whether one carrier kind boxes when its address is
// taken (identity-bearing carriers never do).
func boxable(kind Kind) bool {
	switch kind {
	case KindStruct, KindExternal, KindArray, KindTypeParam, KindInvalid:
		return false
	}
	return true
}

// scanBoxedVars walks one function body (function literals included —
// an inner &x may address an outer variable) and collects every local
// variable whose address is taken. Package-level variables stay out:
// their addresses need the module-level cell story.
func scanBoxedVars(info *types.Info, stmts []ast.Stmt, boxed map[*types.Var]bool) {
	for _, stmt := range stmts {
		ast.Inspect(stmt, func(n ast.Node) bool {
			unary, ok := n.(*ast.UnaryExpr)
			if !ok || unary.Op != token.AND {
				return true
			}
			ident, ok := ast.Unparen(unary.X).(*ast.Ident)
			if !ok {
				return true
			}
			variable, ok := info.Uses[ident].(*types.Var)
			if !ok || variable.Pkg() == nil {
				return true
			}
			if variable.Parent() == variable.Pkg().Scope() {
				return true // package-level variable
			}
			boxed[variable] = true
			return true
		})
	}
}

// boxedVar resolves an identifier to its boxed variable, when it is one.
func (b *builder) boxedVar(ident *ast.Ident) (*types.Var, bool) {
	variable, ok := b.info.Uses[ident].(*types.Var)
	if !ok {
		variable, ok = b.info.Defs[ident].(*types.Var)
	}
	if !ok || !b.boxed[variable] {
		return nil, false
	}
	return variable, true
}

// prependBoxedParams declares the cells of address-taken parameters at
// the top of the body: each cell captures the incoming value once.
func prependBoxedParams(body *Block, boxedParams []Var) {
	if len(boxedParams) == 0 {
		return
	}
	cells := make([]Stmt, 0, len(boxedParams))
	for _, param := range boxedParams {
		cells = append(cells, &BoxedDecl{Cell: cellName(param.Name), Elem: param.Type,
			Init: &VarRef{Name: param.Name, T: param.Type}})
	}
	body.Stmts = append(cells, body.Stmts...)
}

// boxDeclaredNames rewrites the declaration of any address-taken name
// into its cell declaration, preserving source order and evaluation.
// Tuple-bound names cannot box yet and fail closed.
func (b *builder) boxDeclaredNames(idents []ast.Expr, decl *DeclStmt, span Span) (Stmt, error) {
	anyBoxed := false
	for _, lhs := range idents {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		if variable, isBoxed := b.boxedVar(ident); isBoxed {
			t, err := b.typeOf(variable.Type(), span)
			if err == nil && boxable(t.Kind) {
				anyBoxed = true
			}
		}
	}
	if !anyBoxed {
		return decl, nil
	}
	if decl.Tuple != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "address of a tuple-bound variable", Span: span}
	}
	out := &StmtSeq{}
	for i, name := range decl.Names {
		single := &DeclStmt{Names: []string{name}, Types: []Type{decl.Types[i]}, Values: []Expr{decl.Values[i]}}
		ident, isIdent := idents[i].(*ast.Ident)
		if isIdent && name != "_" {
			if _, isBoxed := b.boxedVar(ident); isBoxed && boxable(decl.Types[i].Kind) {
				b.use("boxedDecl")
				out.Stmts = append(out.Stmts, &BoxedDecl{Cell: cellName(name), Elem: decl.Types[i], Init: decl.Values[i]})
				continue
			}
		}
		out.Stmts = append(out.Stmts, single)
	}
	return out, nil
}
