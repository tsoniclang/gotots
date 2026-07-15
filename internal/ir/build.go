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
	fset      *token.FileSet
	info      *types.Info
	pkgPath   string
	sourceDir string
	// unit is the set of package paths translated together; named types,
	// calls, and methods resolve across it, and everything outside fails
	// closed.
	unit       Scope
	operations map[string]bool
	// sites collects every unsupported operation of the outermost body,
	// shared with closure child builders so nested findings bubble up.
	sites      *[]UnsupportedSite
	deferCount int
	// results are the enclosing function's result types, giving return
	// expressions their expected types.
	results []Type
	// namedResults, when set, are the enclosing function's named results:
	// zero-initialized locals that bare returns return.
	namedResults []Var
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

// BuildFunc converts one typed top-level function or method declaration
// into IR. unit is the set of co-translated package paths; bodyHash is the
// census body hash for proof-chain linkage.
func BuildFunc(p *packages.Package, sourceDir string, unit Scope, decl *ast.FuncDecl, id, bodyHash string) (*Func, error) {
	b := &builder{
		fset:       p.Fset,
		info:       p.TypesInfo,
		pkgPath:    p.PkgPath,
		sourceDir:  sourceDir,
		unit:       unit,
		operations: map[string]bool{},
		sites:      &[]UnsupportedSite{},
	}
	span := b.span(decl.Pos())
	object, ok := b.info.Defs[decl.Name].(*types.Func)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "function without typed definition", Span: span}
	}
	// A variadic function's final parameter is exactly a slice: call
	// sites pack trailing arguments (or pass nil, or the spread slice
	// itself), so the declaration needs no special shape.
	signature := object.Type().(*types.Signature)

	function := &Func{
		ID:       id,
		Package:  p.PkgPath,
		Name:     decl.Name.Name,
		Exported: decl.Name.IsExported(),
		Span:     span,
		BodyHash: bodyHash,
	}
	// A declaration-level finding makes the whole body unimplemented:
	// its one site is recorded and the body is not built (its types are
	// unavailable or its effects cannot be represented).
	declarationSite := func(err error) (*Func, error) {
		if !b.recordSite(err) {
			return nil, err
		}
		return b.finalize(function), nil
	}
	if decl.Body == nil {
		return declarationSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "bodyless function", Span: span})
	}
	if decl.Recv == nil && decl.Name.Name == "init" {
		// Package initializers need the initialization subsystem (import
		// DAG order, once semantics); emitting them as ordinary functions
		// would silently drop their effects.
		return declarationSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "package init function (initialization order subsystem)", Span: span})
	}
	if decl.Type.TypeParams != nil {
		names, err := b.admitGenericFunction(object, span)
		if err != nil {
			return declarationSite(err)
		}
		function.TypeParams = names
	}

	if recv := signature.Recv(); recv != nil {
		// A pointer receiver binds the class instance; a value receiver
		// binds a clone on entry, so receiver mutations never reach the
		// caller — Go's receiver copy exactly.
		if recv.Name() == "" || recv.Name() == "_" {
			return declarationSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "unnamed or blank receiver", Span: span})
		}
		recvType, err := b.typeOf(recv.Type(), span)
		if err != nil {
			return declarationSite(err)
		}
		function.Receiver = &Var{Name: recv.Name(), Type: recvType}
	}

	params := signature.Params()
	for i := range params.Len() {
		parameter := params.At(i)
		if parameter.Name() == "" || parameter.Name() == "_" {
			return declarationSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "unnamed or blank parameter", Span: span})
		}
		t, err := b.typeOf(parameter.Type(), span)
		if err != nil {
			return declarationSite(err)
		}
		function.Params = append(function.Params, Var{Name: parameter.Name(), Type: t})
	}
	results := signature.Results()
	for i := range results.Len() {
		result := results.At(i)
		if result.Name() == "_" {
			return declarationSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: "blank named result", Span: span})
		}
		t, err := b.typeOf(result.Type(), span)
		if err != nil {
			return declarationSite(err)
		}
		function.Results = append(function.Results, Var{Name: result.Name(), Type: t})
		b.results = append(b.results, t)
		if result.Name() != "" {
			b.namedResults = append(b.namedResults, Var{Name: result.Name(), Type: t})
		}
	}

	body, err := b.buildTopLevel(decl.Body.List)
	if err != nil {
		return nil, err
	}
	// Named results are zero-initialized locals declared before the body.
	if err := b.prependNamedResultZeros(body, span); err != nil {
		return declarationSite(err)
	}
	function.Body = body
	return b.finalize(function), nil
}

// finalize assigns the body's support state: any recorded site makes it
// unimplemented, its body is withheld (never emitted), and every site
// stays on the record.
func (b *builder) finalize(function *Func) *Func {
	for operation := range b.operations {
		function.Operations = append(function.Operations, operation)
	}
	sort.Strings(function.Operations)
	function.Sites = *b.sites
	if len(function.Sites) > 0 {
		function.Support = SupportUnimplemented
		function.Body = nil
	} else {
		function.Support = SupportGenerated
	}
	return function
}

// buildTopLevel builds the function's top-level statement list, lowering
// defers into nested try/finally: everything after a defer becomes the
// try body, the deferred call (with its receiver and arguments captured
// at the defer site) becomes the finally. Recursion yields Go's LIFO
// order. Defers below the top level run at function exit, which block
// nesting cannot express — they fail closed until the general defer-stack
// lowering exists.
func (b *builder) buildTopLevel(stmts []ast.Stmt) (*Block, error) {
	out := &Block{}
	for index, stmt := range stmts {
		deferStmt, isDefer := stmt.(*ast.DeferStmt)
		if !isDefer {
			built, err := b.buildStmt(stmt)
			if err != nil {
				if b.recordSite(err) {
					out.Stmts = append(out.Stmts, &UnimplementedStmt{Site: (*b.sites)[len(*b.sites)-1]})
					continue
				}
				return nil, err
			}
			out.Stmts = append(out.Stmts, built)
			continue
		}

		if len(b.namedResults) > 0 {
			// A deferred call can observe and mutate named results after
			// the return values are set; try/finally cannot express that
			// visibility, so the combination fails closed. Accounting
			// continues over the remaining statements.
			b.recordSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
				Construct: "defer in a function with named results (deferred result mutation)", Span: b.span(deferStmt.Pos())})
			out.Stmts = append(out.Stmts, &UnimplementedStmt{Site: (*b.sites)[len(*b.sites)-1]})
			continue
		}
		captures, deferredCall, err := b.buildDeferredCall(deferStmt)
		if err != nil {
			if b.recordSite(err) {
				out.Stmts = append(out.Stmts, &UnimplementedStmt{Site: (*b.sites)[len(*b.sites)-1]})
				continue
			}
			return nil, err
		}
		rest, err := b.buildTopLevel(stmts[index+1:])
		if err != nil {
			return nil, err
		}
		out.Stmts = append(out.Stmts, captures...)
		out.Stmts = append(out.Stmts, &TryFinally{
			Body:    rest,
			Finally: &Block{Stmts: []Stmt{&ExprStmt{Call: deferredCall}}},
		})
		b.use("defer")
		return out, nil
	}
	return out, nil
}

// buildDeferredCall captures the deferred call's receiver and arguments at
// the defer site (Go evaluates them when the defer statement executes) and
// returns the capture declarations plus the call over the captures.
func (b *builder) buildDeferredCall(deferStmt *ast.DeferStmt) ([]Stmt, Expr, error) {
	span := b.span(deferStmt.Pos())
	built, err := b.buildAnyCall(deferStmt.Call)
	if err != nil {
		return nil, nil, err
	}
	prefix := fmt.Sprintf("_d%d$", b.deferCount)
	b.deferCount++

	var captures []Stmt
	capture := func(name string, value Expr) *VarRef {
		captures = append(captures, &DeclStmt{
			Names:  []string{name},
			Types:  []Type{value.Type()},
			Values: []Expr{value},
		})
		return &VarRef{Name: name, T: value.Type()}
	}

	switch call := built.(type) {
	case *Call:
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *MethodCall:
		// A value receiver copies when the defer statement evaluates —
		// mutations between the defer and the call never reach it — so a
		// struct-valued receiver captures a clone.
		recv := call.Recv
		if !call.PointerRecv {
			recv = b.bindStructValue(recv)
		}
		call.Recv = capture(prefix+"_r", recv)
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	case *DynCall:
		// The function value is captured at the defer site; a nil value
		// panics when invoked, not when deferred — exactly Go.
		call.Fun = capture(prefix+"_f", call.Fun)
		for i, arg := range call.Args {
			call.Args[i] = capture(fmt.Sprintf("%s_a%d", prefix, i), arg)
		}
		return captures, call, nil
	}
	return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "deferred non-call expression", Span: span}
}

// buildBlock builds one statement list, recovering per statement: an
// unsupported operation records its exact site and the walk continues,
// so one finding never hides later unsupported operations. A body with
// any site is unimplemented and never emitted.
func (b *builder) buildBlock(block *ast.BlockStmt) (*Block, error) {
	out := &Block{}
	for _, stmt := range block.List {
		built, err := b.buildStmt(stmt)
		if err != nil {
			if b.recordSite(err) {
				out.Stmts = append(out.Stmts, &UnimplementedStmt{Site: (*b.sites)[len(*b.sites)-1]})
				continue
			}
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

	case *ast.RangeStmt:
		return b.buildRange(n)

	case *ast.SwitchStmt:
		return b.buildSwitch(n)

	case *ast.TypeSwitchStmt:
		return b.buildTypeSwitch(n)

	case *ast.ReturnStmt:
		return b.buildReturn(n)

	case *ast.DeferStmt:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
			Construct: "defer below the function's top-level block (runs at function exit; needs the defer-stack lowering)", Span: span}

	case *ast.ExprStmt:
		call, ok := n.X.(*ast.CallExpr)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("expression statement %T", n.X), Span: span}
		}
		if _, isConversion := b.conversionTarget(call); isConversion {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "conversion as statement", Span: span}
		}
		if builtin, isBuiltin := b.builtinCallee(call); isBuiltin {
			switch builtin.Name() {
			case "delete":
				return b.buildMapDelete(call)
			case "clear":
				return b.buildClear(call)
			case "panic":
				return b.buildPanic(call)
			case "copy":
				// copy for effect: the returned count is discarded.
				built, err := b.buildBuiltin(call, builtin, nil)
				if err != nil {
					return nil, err
				}
				b.use("exprStmt")
				return &ExprStmt{Call: built}, nil
			}
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "builtin statement " + builtin.Name(), Span: span}
		}
		built, err := b.buildAnyCall(call)
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
