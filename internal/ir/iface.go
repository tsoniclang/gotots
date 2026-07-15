package ir

import (
	"go/ast"
	"go/types"
	"strings"
)

// rttiFor resolves the shared rtti reference of one concrete Go type:
// a predeclared basic type, a named type (struct or carrier), or a
// pointer to a named struct. Every other dynamic type fails closed.
func (b *builder) rttiFor(t types.Type, span Span) (RttiRef, error) {
	switch concrete := types.Unalias(t).(type) {
	case *types.Basic:
		// Only true predeclared types reach here (named carriers are
		// *types.Named). byte and rune are aliases: their dynamic types
		// are uint8 and int32, so names canonicalize by basic kind.
		name, ok := predeclaredRttiName(concrete.Kind())
		if !ok {
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of type " + t.String(), Span: span}
		}
		return RttiRef{Predeclared: name}, nil

	case *types.Named:
		obj := concrete.Obj()
		if obj.Pkg() == nil || !b.unit.Owns(obj.Pkg().Path()) {
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of a type outside the translated unit", Span: span}
		}
		return RttiRef{Pkg: obj.Pkg().Path(), TypeName: obj.Name()}, nil

	case *types.Pointer:
		named, ok := types.Unalias(concrete.Elem()).(*types.Named)
		if !ok {
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of a pointer to an unnamed type", Span: span}
		}
		obj := named.Obj()
		if obj.Pkg() == nil || !b.unit.Owns(obj.Pkg().Path()) {
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of a type outside the translated unit", Span: span}
		}
		return RttiRef{Pkg: obj.Pkg().Path(), TypeName: obj.Name(), Pointer: true}, nil
	}
	return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
		Construct: "interface value of type " + t.String(), Span: span}
}

// displayOf spells a type the way Go's runtime messages do: package
// names (not paths) qualify named types, and the empty interface prints
// with the runtime's spacing.
func displayOf(t types.Type) string {
	spelled := types.TypeString(t, func(p *types.Package) string { return p.Name() })
	if spelled == "any" {
		return "interface {}"
	}
	return strings.ReplaceAll(spelled, "interface{}", "interface {}")
}

// predeclaredRttiName canonicalizes a predeclared basic kind onto its
// ABI rtti name.
func predeclaredRttiName(kind types.BasicKind) (string, bool) {
	switch kind {
	case types.Bool:
		return "bool", true
	case types.String:
		return "string", true
	case types.Int:
		return "int", true
	case types.Int8:
		return "int8", true
	case types.Int16:
		return "int16", true
	case types.Int32:
		return "int32", true
	case types.Int64:
		return "int64", true
	case types.Uint:
		return "uint", true
	case types.Uint8:
		return "uint8", true
	case types.Uint16:
		return "uint16", true
	case types.Uint32:
		return "uint32", true
	case types.Uint64:
		return "uint64", true
	case types.Uintptr:
		return "uintptr", true
	case types.Float32:
		return "float32", true
	case types.Float64:
		return "float64", true
	}
	return "", false
}

// boxIfaceValue converts a concrete expression to an interface value at
// a binding site. Interface-to-interface bindings pass the box through;
// nil stays the nil interface; struct values are copied into the box.
func (b *builder) boxIfaceValue(built Expr, source types.Type, expected Type, span Span) (Expr, error) {
	if built.Type().Kind == KindIface {
		return built, nil
	}
	if _, isNil := built.(*NilConst); isNil {
		return built, nil
	}
	rtti, err := b.rttiFor(source, span)
	if err != nil {
		return nil, err
	}
	b.use("ifaceBox")
	return &IfaceBox{X: b.bindStructValue(built), Rtti: rtti, T: expected}, nil
}

// buildIfaceMethodCall dispatches a method through an interface value's
// method table (a nil interface panics exactly like a nil dereference).
func (b *builder) buildIfaceMethodCall(n *ast.CallExpr, recv Expr, method *types.Func) (Expr, error) {
	signature := method.Type().(*types.Signature)
	out := &IfaceCall{Recv: recv, Method: method.Name()}
	if err := b.buildCallArgsResults(n, signature, &out.Args, &out.Results); err != nil {
		return nil, err
	}
	b.use("ifaceCall")
	return out, nil
}

// buildTypeAssert lowers x.(T) for a concrete reviewed target type.
func (b *builder) buildTypeAssert(n *ast.TypeAssertExpr, commaOk bool) (Expr, error) {
	span := b.span(n.Pos())
	operand, err := b.buildExpr(n.X)
	if err != nil {
		return nil, err
	}
	if operand.Type().Kind != KindIface {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "type assertion on " + operand.Type().Go, Span: span}
	}
	targetGoType := b.info.Types[n.Type].Type
	if _, isIface := targetGoType.Underlying().(*types.Interface); isIface {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "type assertion to an interface type (method-set test)", Span: span}
	}
	target, err := b.typeOf(targetGoType, span)
	if err != nil {
		return nil, err
	}
	rtti, err := b.rttiFor(targetGoType, span)
	if err != nil {
		return nil, err
	}
	b.use("typeAssert")
	return &TypeAssert{
		X:             operand,
		Target:        target,
		Rtti:          rtti,
		SourceDisplay: displayOf(b.info.Types[n.X].Type),
		CommaOk:       commaOk,
	}, nil
}

// buildTypeSwitch lowers a type switch onto rtti identity tests in
// clause order.
func (b *builder) buildTypeSwitch(n *ast.TypeSwitchStmt) (Stmt, error) {
	span := b.span(n.Pos())
	out := &TypeSwitchStmt{}
	if n.Init != nil {
		init, err := b.buildStmt(n.Init)
		if err != nil {
			return nil, err
		}
		out.Init = init
	}

	// The guard is either `x.(type)` or `y := x.(type)`.
	var assertion *ast.TypeAssertExpr
	switch guard := n.Assign.(type) {
	case *ast.ExprStmt:
		assertion = guard.X.(*ast.TypeAssertExpr)
	case *ast.AssignStmt:
		bind := guard.Lhs[0].(*ast.Ident)
		if bind.Name != "_" {
			out.Bind = bind.Name
		}
		assertion = guard.Rhs[0].(*ast.TypeAssertExpr)
	default:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "type switch guard form", Span: span}
	}
	operand, err := b.buildExpr(assertion.X)
	if err != nil {
		return nil, err
	}
	if operand.Type().Kind != KindIface {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "type switch on " + operand.Type().Go, Span: span}
	}
	out.X = operand

	for _, clauseStmt := range n.Body.List {
		clause := clauseStmt.(*ast.CaseClause)
		built := TypeSwitchClause{BindType: operand.Type()}
		for _, expr := range clause.List {
			if tv, ok := b.info.Types[expr]; ok && tv.IsNil() {
				built.Targets = append(built.Targets, TypeSwitchTarget{Nil: true})
				continue
			}
			targetGoType := b.info.Types[expr].Type
			if _, isIface := targetGoType.Underlying().(*types.Interface); isIface {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
					Construct: "type switch clause with an interface type (method-set test)", Span: b.span(expr.Pos())}
			}
			target, err := b.typeOf(targetGoType, b.span(expr.Pos()))
			if err != nil {
				return nil, err
			}
			rtti, err := b.rttiFor(targetGoType, b.span(expr.Pos()))
			if err != nil {
				return nil, err
			}
			built.Targets = append(built.Targets, TypeSwitchTarget{Rtti: rtti, Target: target})
		}
		// A single concrete clause binds the unboxed value in that type;
		// nil, multi-type, and default clauses bind the interface value.
		if len(built.Targets) == 1 && !built.Targets[0].Nil {
			built.BindType = built.Targets[0].Target
		}
		body, err := b.buildBlock(&ast.BlockStmt{List: clause.Body})
		if err != nil {
			return nil, err
		}
		built.Body = body
		out.Clauses = append(out.Clauses, built)
	}
	b.use("typeSwitch")
	return out, nil
}
