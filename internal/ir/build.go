package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// builder carries the typed context for one function body.
type builder struct {
	fset       *token.FileSet
	info       *types.Info
	pkgPath    string
	sourceDir  string
	operations map[string]bool
}

func (b *builder) span(pos token.Pos) Span {
	position := b.fset.Position(pos)
	file := position.Filename
	if relative, err := filepath.Rel(b.sourceDir, file); err == nil && !strings.HasPrefix(relative, "..") {
		file = filepath.ToSlash(relative)
	}
	return Span{File: file, Line: position.Line, Col: position.Column}
}

func (b *builder) use(operation string) { b.operations[operation] = true }

// BuildFunc converts one typed top-level function declaration into IR.
// bodyHash is the census body hash for proof-chain linkage.
func BuildFunc(p *packages.Package, sourceDir string, decl *ast.FuncDecl, id, bodyHash string) (*Func, error) {
	b := &builder{
		fset:       p.Fset,
		info:       p.TypesInfo,
		pkgPath:    p.PkgPath,
		sourceDir:  sourceDir,
		operations: map[string]bool{},
	}
	span := b.span(decl.Pos())
	if decl.Recv != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "method declaration", Span: span}
	}
	if decl.Type.TypeParams != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "generic function", Span: span}
	}
	if decl.Body == nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "bodyless function", Span: span}
	}
	object, ok := b.info.Defs[decl.Name].(*types.Func)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "function without typed definition", Span: span}
	}
	signature := object.Type().(*types.Signature)
	if signature.Variadic() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "variadic function", Span: span}
	}

	function := &Func{
		ID:       id,
		Package:  p.PkgPath,
		Name:     decl.Name.Name,
		Exported: decl.Name.IsExported(),
		Span:     span,
		BodyHash: bodyHash,
	}
	params := signature.Params()
	for i := range params.Len() {
		parameter := params.At(i)
		if parameter.Name() == "" || parameter.Name() == "_" {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "unnamed or blank parameter", Span: span}
		}
		t, err := typeOf(parameter.Type(), span)
		if err != nil {
			return nil, err
		}
		function.Params = append(function.Params, Var{Name: parameter.Name(), Type: t})
	}
	results := signature.Results()
	for i := range results.Len() {
		result := results.At(i)
		if result.Name() != "" {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "named result", Span: span}
		}
		t, err := typeOf(result.Type(), span)
		if err != nil {
			return nil, err
		}
		function.Results = append(function.Results, Var{Type: t})
	}

	body, err := b.buildBlock(decl.Body)
	if err != nil {
		return nil, err
	}
	function.Body = body
	for operation := range b.operations {
		function.Operations = append(function.Operations, operation)
	}
	sort.Strings(function.Operations)
	return function, nil
}

func (b *builder) buildBlock(block *ast.BlockStmt) (*Block, error) {
	out := &Block{}
	for _, stmt := range block.List {
		built, err := b.buildStmt(stmt)
		if err != nil {
			return nil, err
		}
		out.Stmts = append(out.Stmts, built)
	}
	return out, nil
}

func (b *builder) buildStmt(stmt ast.Stmt) (Stmt, error) {
	span := b.span(stmt.Pos())
	switch n := stmt.(type) {
	case *ast.BlockStmt:
		return b.buildBlock(n)

	case *ast.DeclStmt:
		return b.buildDeclStmt(n)

	case *ast.AssignStmt:
		return b.buildAssign(n)

	case *ast.IncDecStmt:
		return b.buildIncDec(n)

	case *ast.IfStmt:
		return b.buildIf(n)

	case *ast.ForStmt:
		return b.buildFor(n)

	case *ast.ReturnStmt:
		return b.buildReturn(n)

	case *ast.ExprStmt:
		call, ok := n.X.(*ast.CallExpr)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("expression statement %T", n.X), Span: span}
		}
		if _, isConversion := b.conversionTarget(call); isConversion {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "conversion as statement", Span: span}
		}
		built, err := b.buildCall(call)
		if err != nil {
			return nil, err
		}
		b.use("exprStmt")
		return &ExprStmt{Call: built}, nil

	case *ast.BranchStmt:
		if n.Label != nil || (n.Tok != token.BREAK && n.Tok != token.CONTINUE) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "branch " + n.Tok.String(), Span: span}
		}
		b.use("branch:" + n.Tok.String())
		return &BranchStmt{Tok: n.Tok}, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("%T", stmt), Span: span}
}

func (b *builder) buildDeclStmt(n *ast.DeclStmt) (Stmt, error) {
	span := b.span(n.Pos())
	decl, ok := n.Decl.(*ast.GenDecl)
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
			t, err := typeOf(object.Type(), span)
			if err != nil {
				return nil, err
			}
			out.Names = append(out.Names, name.Name)
			out.Types = append(out.Types, t)
			if i < len(value.Values) {
				built, err := b.buildExpr(value.Values[i])
				if err != nil {
					return nil, err
				}
				out.Values = append(out.Values, built)
			} else {
				out.Values = append(out.Values, zeroValue(t))
			}
		}
		if len(value.Values) != 0 && len(value.Values) != len(value.Names) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "multi-value var initializer", Span: span}
		}
	}
	b.use("varDecl")
	return out, nil
}

func (b *builder) buildIncDec(n *ast.IncDecStmt) (Stmt, error) {
	span := b.span(n.Pos())
	target, ok := n.X.(*ast.Ident)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "inc/dec of non-variable", Span: span}
	}
	x, err := b.buildExpr(target)
	if err != nil {
		return nil, err
	}
	op := token.ADD
	if n.Tok == token.DEC {
		op = token.SUB
	}
	one := &Const{T: x.Type(), Value: "1"}
	if !x.Type().Kind.Integer() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "inc/dec of " + x.Type().Go, Span: span}
	}
	b.use("incDec")
	return &AssignStmt{Targets: []string{target.Name}, Values: []Expr{&Binary{Op: op, L: x, R: one, T: x.Type()}}}, nil
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
	body, err := b.buildBlock(n.Body)
	if err != nil {
		return nil, err
	}
	out.Body = body
	b.use("for")
	return out, nil
}

func (b *builder) buildReturn(n *ast.ReturnStmt) (Stmt, error) {
	span := b.span(n.Pos())
	if len(n.Results) == 0 {
		b.use("return")
		return &ReturnStmt{}, nil
	}
	// A single multi-result call forwarded as the complete result list.
	if len(n.Results) == 1 {
		if callAST, ok := ast.Unparen(n.Results[0]).(*ast.CallExpr); ok {
			if _, isConversion := b.conversionTarget(callAST); !isConversion {
				if tuple, ok := b.info.Types[callAST].Type.(*types.Tuple); ok && tuple.Len() > 1 {
					call, err := b.buildCall(callAST)
					if err != nil {
						return nil, err
					}
					b.use("return")
					return &ReturnStmt{CallValue: call}, nil
				}
			}
		}
	}
	out := &ReturnStmt{}
	for _, result := range n.Results {
		built, err := b.buildExpr(result)
		if err != nil {
			return nil, err
		}
		out.Values = append(out.Values, built)
	}
	_ = span
	b.use("return")
	return out, nil
}

// zeroValue materializes the Go zero value for a reviewed type.
func zeroValue(t Type) Expr {
	switch {
	case t.Kind == KindBool:
		return &Const{T: t, Value: "false"}
	case t.Kind == KindString:
		return &Const{T: t, Value: `""`}
	default:
		return &Const{T: t, Value: "0"}
	}
}
