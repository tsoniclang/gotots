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

	"github.com/tsoniclang/gotots/internal/typeid"
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
	// binders resolves the type parameters of the declaration being built
	// (a generic function's own, or a generic type's), so component types
	// referencing them canonicalize instead of failing closed.
	binders *typeid.Binders
	// sites collects every unsupported operation of the outermost body,
	// shared with closure child builders so nested findings bubble up.
	sites      *[]UnsupportedSite
	deferCount int
	// results are the enclosing function's result types, giving return
	// expressions their expected types; resultGoTypes keeps the parallel
	// go/types results for tuple-return assignability conversions.
	results       []Type
	resultGoTypes []types.Type
	// namedResults, when set, are the enclosing function's named results:
	// zero-initialized locals that bare returns return.
	namedResults []Var
	// typeSwitchDepth counts enclosing type-switch clauses with no loop
	// or switch between them and the current statement: a break there
	// exits the type switch, which the if/else lowering cannot express
	// yet, so it fails closed.
	typeSwitchDepth int
	// useDeferStack switches every defer in this body onto the LIFO
	// defer stack — set when any defer sits below the top-level block,
	// where try/finally nesting cannot express function-exit timing.
	useDeferStack bool
	// boxed marks every local variable whose address is taken; boxable
	// carriers among them live in mutable cells (shared with closure
	// child builders — an inner &x may address an outer variable).
	boxed map[*types.Var]bool
	// genericObj is the generic function being built, when there is one:
	// type-parameter admissions (map keys) consult its closed-world
	// instantiation evidence.
	genericObj *types.Func
	// genericTypeObj is the generic named type whose method is being
	// built; its instantiation evidence admits receiver type parameters.
	genericTypeObj *types.Named
	// typeSwitchLabels carries one synthesized-label slot per enclosing
	// type switch under construction; a direct break fills the innermost.
	typeSwitchLabels []*string
	// rangeFuncDepth counts enclosing range-over-func bodies: labeled
	// branches cannot cross the yield-closure boundary.
	rangeFuncDepth int
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
	if signature.Recv() != nil {
		mb, err := typeid.MethodBinders(object)
		if err != nil {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION", Construct: err.Error(), Span: span}
		}
		b.binders = mb
	} else {
		b.binders = typeid.FuncBinders(signature)
	}

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
	if decl.Type.TypeParams != nil {
		names, err := b.admitGenericFunction(object, span)
		if err != nil {
			return declarationSite(err)
		}
		function.TypeParams = names
		b.genericObj = object
	}

	if recv := signature.Recv(); recv != nil {
		_, function.PointerReceiver = recv.Type().(*types.Pointer)
		if recvParams := signature.RecvTypeParams(); recvParams != nil {
			// A method on a generic type: one generic function per
			// method, admitted under the type's instantiation evidence
			// (verified at the type declaration).
			for i := range recvParams.Len() {
				function.TypeParams = append(function.TypeParams, recvParams.At(i).Obj().Name())
			}
			recvType := signature.Recv().Type()
			if pointer, isPointer := recvType.(*types.Pointer); isPointer {
				recvType = pointer.Elem()
			}
			if named, isNamed := types.Unalias(recvType).(*types.Named); isNamed {
				b.genericTypeObj = named
			}
		}
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

	if b.boxed == nil {
		b.boxed = map[*types.Var]bool{}
	}
	scanBoxedVars(b.info, decl.Body.List, b.boxed)
	var boxedParams []Var
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
		if b.boxed[parameter] && boxable(t.Kind) {
			boxedParams = append(boxedParams, Var{Name: parameter.Name(), Type: t})
		}
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
		b.resultGoTypes = append(b.resultGoTypes, result.Type())
		if result.Name() != "" {
			if result.Parent() != nil && b.namedResultAddressed(result) {
				return declarationSite(&Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "address of a named result", Span: span})
			}
			b.namedResults = append(b.namedResults, Var{Name: result.Name(), Type: t})
		}
	}

	b.useDeferStack = hasNestedDefer(decl.Body.List)
	body, err := b.buildTopLevel(decl.Body.List)
	if err != nil {
		return nil, err
	}
	// Named results are zero-initialized locals declared before the body.
	if err := b.prependNamedResultZeros(body, span); err != nil {
		return declarationSite(err)
	}
	prependBoxedParams(body, boxedParams)
	function.Body = body
	function.UsesDeferStack = b.useDeferStack
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
		function.SlicePlans = AnalyzeSlicePlans(function)
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
		if b.useDeferStack {
			// Uniform LIFO through the defer stack: a top-level defer
			// pushed before a nested one runs after it, exactly Go.
			captures, deferredCall, err := b.buildDeferredCall(deferStmt)
			if err != nil {
				if b.recordSite(err) {
					out.Stmts = append(out.Stmts, &UnimplementedStmt{Site: (*b.sites)[len(*b.sites)-1]})
					continue
				}
				return nil, err
			}
			out.Stmts = append(out.Stmts, captures...)
			out.Stmts = append(out.Stmts, &DeferPush{Call: deferredCall})
			b.use("defer:stack")
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

// buildBlock builds one statement list, recovering per statement: an
// unsupported operation records its exact site and the walk continues,
// so one finding never hides later unsupported operations. A body with
// any site is unimplemented and never emitted.
// buildBreakableBody builds a body owning break (loop or switch),
// clearing the type-switch break restriction.
func (b *builder) buildBreakableBody(body *ast.BlockStmt) (*Block, error) {
	saved := b.typeSwitchDepth
	b.typeSwitchDepth = 0
	out, err := b.buildBlock(body)
	b.typeSwitchDepth = saved
	return out, err
}

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
		if !b.useDeferStack {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
				Construct: "defer below the function's top-level block (runs at function exit; needs the defer-stack lowering)", Span: span}
		}
		if len(b.namedResults) > 0 {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
				Construct: "defer in a function with named results (deferred result mutation)", Span: span}
		}
		captures, deferredCall, err := b.buildDeferredCall(n)
		if err != nil {
			return nil, err
		}
		b.use("defer:stack")
		return &StmtSeq{Stmts: append(captures, &DeferPush{Call: deferredCall})}, nil

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
		if n.Tok != token.BREAK && n.Tok != token.CONTINUE {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "branch " + n.Tok.String(), Span: span}
		}
		label := ""
		if n.Label != nil {
			if b.rangeFuncDepth > 0 {
				// A labeled branch cannot cross the yield-closure boundary.
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "labeled branch inside a range-over-func body", Span: span}
			}
			label = n.Label.Name
		}
		if label == "" && n.Tok == token.BREAK && b.typeSwitchDepth > 0 {
			// break exits the type switch: the if/else lowering routes it
			// through the type switch's synthesized labeled block.
			if len(b.typeSwitchLabels) == 0 {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "break inside a type switch clause", Span: span}
			}
			slot := b.typeSwitchLabels[len(b.typeSwitchLabels)-1]
			if *slot == "" {
				*slot = fmt.Sprintf("ts%d$", len(b.typeSwitchLabels))
			}
			b.use("branch:break")
			return &BranchStmt{Tok: token.BREAK, Label: *slot}, nil
		}
		b.use("branch:" + n.Tok.String())
		return &BranchStmt{Tok: n.Tok, Label: label}, nil

	case *ast.LabeledStmt:
		switch n.Stmt.(type) {
		case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		default:
			// Labels exist for branch targets; a label on any other
			// statement implies goto, which stays out.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "label on a non-loop statement", Span: span}
		}
		inner, err := b.buildStmt(n.Stmt)
		if err != nil {
			return nil, err
		}
		if _, isRangeFunc := inner.(*RangeFunc); isRangeFunc {
			// The sequence-call lowering is not a loop statement; its
			// branches already fail closed inside the yield closure.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "label on a range-over-func loop", Span: span}
		}
		b.use("label")
		return &LabeledStmt{Label: n.Label.Name, Stmt: inner}, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("%T", stmt), Span: span}
}
