package ir

import (
	"fmt"
	"go/ast"
	"go/types"
)

// buildClosure lowers a function literal into a Closure. The child
// builder shares the typed context and the operation record but carries
// its own per-function state (results, defers). Captured variables
// reference the enclosing scope directly: Go and JS both capture by
// reference, and both give loop variables per-iteration bindings, so
// capture semantics coincide.
func (b *builder) buildClosure(lit *ast.FuncLit) (Expr, error) {
	span := b.span(lit.Pos())
	t, err := b.typeOf(b.info.Types[lit].Type, span)
	if err != nil {
		return nil, err
	}
	signature := b.info.Types[lit].Type.Underlying().(*types.Signature)

	child := &builder{
		fset:       b.fset,
		info:       b.info,
		pkgPath:    b.pkgPath,
		sourceDir:  b.sourceDir,
		unit:       b.unit,
		operations: b.operations,
		sites:      b.sites,
		boxed:      b.boxed,
		genericObj: b.genericObj,
		binders:    b.binders,
	}
	out := &Closure{T: t}
	var boxedParams []Var
	params := signature.Params()
	for i := range params.Len() {
		parameter := params.At(i)
		name := parameter.Name()
		if name == "" || name == "_" {
			// A discarded parameter binds a synthetic name no Go source
			// can spell or reference.
			name = fmt.Sprintf("$discard%d", i)
		}
		parameterType, err := child.typeOf(parameter.Type(), span)
		if err != nil {
			return nil, err
		}
		out.Params = append(out.Params, Var{Name: name, Type: parameterType})
		if child.boxed[parameter] && boxable(parameterType.Kind) {
			boxedParams = append(boxedParams, Var{Name: name, Type: parameterType})
		}
	}
	results := signature.Results()
	for i := range results.Len() {
		result := results.At(i)
		if result.Name() == "_" {
			return nil, &Unsupported{Kind: KindBlankNamedResult, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "blank named result", Span: span}
		}
		resultType, err := child.typeOf(result.Type(), span)
		if err != nil {
			return nil, err
		}
		out.Results = append(out.Results, Var{Name: result.Name(), Type: resultType})
		child.results = append(child.results, resultType)
		child.resultGoTypes = append(child.resultGoTypes, result.Type())
		if result.Name() != "" {
			child.namedResults = append(child.namedResults, Var{Name: result.Name(), Type: resultType})
		}
	}

	child.useDeferStack = hasNestedDefer(lit.Body.List)
	body, err := child.buildTopLevel(lit.Body.List)
	if err != nil {
		return nil, err
	}
	if err := child.prependNamedResultZeros(body, span); err != nil {
		return nil, err
	}
	prependBoxedParams(body, boxedParams)
	out.Body = body
	out.UsesDeferStack = child.useDeferStack
	b.use("closure")
	return out, nil
}

// buildFuncRef references a package-level function of the translated
// unit as a first-class value.
func (b *builder) buildFuncRef(function *types.Func, span Span) (Expr, error) {
	if function.Pkg() == nil || !b.unit.Owns(function.Pkg().Path()) {
		return nil, &Unsupported{Kind: KindReferenceToAFunctionOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reference to a function outside the translated unit", Span: span}
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() != nil {
		return nil, &Unsupported{Kind: KindMethodValueBindTimeReceiverCapture, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method value (bind-time receiver capture)", Span: span}
	}
	t, err := b.typeOf(signature, span)
	if err != nil {
		return nil, err
	}
	b.use("funcRef")
	return &FuncRef{Pkg: function.Pkg().Path(), Name: function.Name(), T: t}, nil
}

// prependNamedResultZeros declares the builder's named results as
// zero-initialized locals at the top of the body.
func (b *builder) prependNamedResultZeros(body *Block, span Span) error {
	if len(b.namedResults) == 0 {
		return nil
	}
	declStmt := &DeclStmt{}
	for _, result := range b.namedResults {
		zero, err := zeroValue(result.Type, span)
		if err != nil {
			return err
		}
		declStmt.Names = append(declStmt.Names, result.Name)
		declStmt.Types = append(declStmt.Types, result.Type)
		declStmt.Values = append(declStmt.Values, zero)
	}
	b.use("namedResults")
	body.Stmts = append([]Stmt{declStmt}, body.Stmts...)
	return nil
}
