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
		fset:           b.fset,
		info:           b.info,
		pkgPath:        b.pkgPath,
		sourceDir:      b.sourceDir,
		unit:           b.unit,
		operations:     b.operations,
		sites:          b.sites,
		boxed:          b.boxed,
		bind:           b.bind,
		genericObj:     b.genericObj,
		genericTypeObj: b.genericTypeObj,
		binders:        b.binders,
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
		} else {
			name = child.bindNameVar(parameter, name)
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
		resultName := result.Name()
		if resultName != "" {
			resultName = child.bindNameVar(result, resultName)
		}
		out.Results = append(out.Results, Var{Name: resultName, Type: resultType})
		child.results = append(child.results, resultType)
		child.resultGoTypes = append(child.resultGoTypes, result.Type())
		if result.Name() != "" {
			child.namedResults = append(child.namedResults, Var{Name: resultName, Type: resultType})
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
func (b *builder) buildFuncRef(function *types.Func, ident *ast.Ident, span Span) (Expr, error) {
	if function.Pkg() == nil {
		return nil, &Unsupported{Kind: KindReferenceToAFunctionOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reference to a function outside the translated unit", Span: span}
	}
	if !b.unit.Owns(function.Pkg().Path()) {
		// A NON-generic external function referenced as a value: the
		// typed stub export IS the value (fail-closed until assembly).
		// Generic references stay out — a bare reference cannot carry the
		// stub's factory protocol.
		signature, isSig := function.Type().(*types.Signature)
		if !isSig || (signature.TypeParams() != nil && signature.TypeParams().Len() > 0) || signature.Recv() != nil {
			return nil, &Unsupported{Kind: KindReferenceToAFunctionOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reference to a function outside the translated unit", Span: span}
		}
		t, err := b.typeOf(signature, span)
		if err != nil {
			return nil, err
		}
		b.unit.AddExternalFunc(function)
		b.use("externFuncRef")
		return &FuncRef{Pkg: function.Pkg().Path(), Name: function.Name(), T: t}, nil
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() != nil {
		return nil, &Unsupported{Kind: KindMethodValueBindTimeReceiverCapture, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method value (bind-time receiver capture)", Span: span}
	}
	if signature.TypeParams() != nil && signature.TypeParams().Len() > 0 {
		// An IMPLICITLY INSTANTIATED generic function referenced as a
		// value (core.Identity as a callback): eta-expand — the value is
		// an exactly typed arrow closing over the instantiation's factory
		// derivations.
		if ident == nil {
			return nil, &Unsupported{Kind: KindReferenceToAFunctionOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reference to a generic function without instantiation evidence", Span: span}
		}
		instance, hasInstance := b.info.Instances[ident]
		if !hasInstance || instance.TypeArgs == nil || signature.Variadic() {
			return nil, &Unsupported{Kind: KindReferenceToAFunctionOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reference to a generic function without instantiation evidence", Span: span}
		}
		instantiated := instance.Type
		if instantiated == nil {
			return nil, &Unsupported{Kind: KindReferenceToAFunctionOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "reference to a generic function without instantiation evidence", Span: span}
		}
		t, err := b.typeOf(instantiated, span)
		if err != nil {
			return nil, err
		}
		out := &GenericFuncValue{Pkg: function.Pkg().Path(), Name: function.Name(), T: t}
		for i := range instance.TypeArgs.Len() {
			argIR, err := b.typeOf(instance.TypeArgs.At(i), span)
			if err != nil {
				return nil, err
			}
			out.TypeArgs = append(out.TypeArgs, argIR)
			out.KeyedParams = append(out.KeyedParams, b.unit.ParamRequiresKeyOp(function, i))
			out.HardKeyed = append(out.HardKeyed, b.unit.ParamRequiresSVZKey(function, i))
		}
		if !b.unit.Owns(function.Pkg().Path()) {
			b.unit.AddExternalFunc(function)
		}
		b.use("genericFuncRef")
		return out, nil
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
