package ir

import (
	"go/ast"
	"go/token"
	"go/types"

	"sort"
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
		if obj.Pkg() == nil {
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of type " + t.String(), Span: span}
		}
		if !b.unit.Owns(obj.Pkg().Path()) {
			// An external named type: an interned rtti whose method
			// dispatch routes through the external contracts.
			composite, err := b.canonicalTypeID(t)
			if err != nil {
				return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE",
					Construct: "runtime type identity of " + t.String() + " (an open type parameter has no single runtime type)", Span: span}
			}
			return RttiRef{Composite: composite, Display: displayOf(t),
				ExternID: obj.Pkg().Path() + "." + obj.Name()}, nil
		}
		if concrete.TypeArgs() != nil && concrete.TypeArgs().Len() > 0 {
			// Instantiated generic types have per-instantiation identity;
			// their interned rtti (with a dispatching method table) is a
			// separate lowering.
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of an instantiated generic type", Span: span}
		}
		return RttiRef{Pkg: obj.Pkg().Path(), TypeName: obj.Name()}, nil

	case *types.Pointer:
		named, ok := types.Unalias(concrete.Elem()).(*types.Named)
		if !ok {
			// A pointer to an unnamed type: the interned composite rtti.
			return b.compositeRtti(t, span, "")
		}
		obj := named.Obj()
		if obj.Pkg() == nil {
			return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
				Construct: "interface value of type " + t.String(), Span: span}
		}
		if !b.unit.Owns(obj.Pkg().Path()) {
			return b.compositeRtti(t, span, obj.Pkg().Path()+"."+obj.Name())
		}
		return RttiRef{Pkg: obj.Pkg().Path(), TypeName: obj.Name(), Pointer: true}, nil

	case *types.Slice, *types.Map, *types.Array, *types.Signature:
		return b.compositeRtti(t, span, "")
	}
	return RttiRef{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION",
		Construct: "interface value of type " + t.String(), Span: span}
}

// buildIfaceMethodCall dispatches a method through an interface value's
// method table (a nil interface panics exactly like a nil dereference).
func (b *builder) buildIfaceMethodCall(n *ast.CallExpr, recv Expr, method *types.Func, selector *ast.SelectorExpr) (Expr, error) {
	span := b.span(n.Pos())
	signature := method.Type().(*types.Signature)
	out := &IfaceCall{Recv: recv, Display: method.Name()}
	if err := b.buildCallArgsResults(n, signature, &out.Args, &out.Results); err != nil {
		return nil, err
	}
	// The closed dispatch set lives on the receiver's interface TYPE
	// (its IfaceMembers union); emission switches the literal
	// discriminant — no per-call set, no value imports of implementers.
	_ = span
	_ = selector
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
	if targetIface, isIface := targetGoType.Underlying().(*types.Interface); isIface {
		// x.(I): the dynamic type's method set must carry every method
		// of the target interface; the result is the same boxed value.
		target, err := b.typeOf(targetGoType, span)
		if err != nil {
			return nil, err
		}
		implementers, err := b.resolveImplementerTokens(targetIface, span)
		if err != nil {
			return nil, err
		}
		// x.(interface{}) and other empty-interface targets are
		// universal: EVERY non-nil dynamic value passes, including
		// predeclared and composite tokens the named-implementer sweep
		// cannot enumerate.
		universal := targetIface.NumMethods() == 0
		required := make([]string, 0, targetIface.NumMethods())
		for i := range targetIface.NumMethods() {
			required = append(required, targetIface.Method(i).Name())
		}
		sort.Strings(required)
		b.use("typeAssert:iface")
		return &IfaceAssert{
			X:             operand,
			Target:        target,
			Universal:     universal,
			Implementers:  implementers,
			Required:      required,
			SourceDisplay: displayOf(b.info.Types[n.X].Type),
			TargetDisplay: displayOf(targetGoType),
			CommaOk:       commaOk,
		}, nil
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
		TargetDisplay: displayOf(targetGoType),
		CommaOk:       commaOk,
	}, nil
}

// buildTypeSwitch lowers a type switch onto rtti identity tests in
// clause order.
func (b *builder) buildTypeSwitch(n *ast.TypeSwitchStmt) (Stmt, error) {
	span := b.span(n.Pos())
	out := &TypeSwitchStmt{}
	b.typeSwitchLabels = append(b.typeSwitchLabels, &out.BreakLabel)
	defer func() { b.typeSwitchLabels = b.typeSwitchLabels[:len(b.typeSwitchLabels)-1] }()
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
		if out.Bind != "" {
			// The clause's implicit binding variable (typed to the case)
			// cannot have its address taken: the plain binding is not a
			// cell.
			if implicit, ok := b.info.Implicits[clause].(*types.Var); ok && b.boxed[implicit] {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT",
					Construct: "address of a type-switch variable", Span: b.span(clause.Pos())}
			}
		}
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
		b.typeSwitchDepth++
		body, err := b.buildBlock(&ast.BlockStmt{List: clause.Body})
		b.typeSwitchDepth--
		if err != nil {
			return nil, err
		}
		built.Body = body
		out.Clauses = append(out.Clauses, built)
	}
	b.use("typeSwitch")
	return out, nil
}

// IfaceAssert is x.(I) for an interface target: a method-set test over
// the dynamic type's rtti, returning the same boxed value.
type IfaceAssert struct {
	X      Expr
	Target Type
	// Universal marks an empty-interface target: every non-nil dynamic
	// value passes (predeclared and composite tokens included), so the
	// assert is only the nil test.
	Universal bool
	// Implementers is the closed set of dynamic-type tokens that satisfy
	// the target interface (whole-unit static resolution). The assert is
	// a token-membership test, not a runtime method-set probe. Required
	// is the target's sorted method displays for the miss diagnostic.
	Implementers  []RttiRef
	Required      []string
	SourceDisplay string
	TargetDisplay string
	CommaOk       bool
}

func (*IfaceAssert) expr() {}
func (a *IfaceAssert) Type() Type {
	if a.CommaOk {
		return Type{Kind: KindInvalid, Go: "(iface, bool)"}
	}
	return a.Target
}

// IfaceEqual compares two interface values by Go's rule: equal dynamic
// types and equal values (uncomparable dynamic types panic at runtime).
// Iface is the SUPERTYPE interface both operands are assignable to — the
// union whose generated equality function the comparison narrows over, so
// both boxes are members of it.
type IfaceEqual struct {
	L, R   Expr
	Iface  Type
	Negate bool
}

// IfaceConcreteEqual compares an interface value against a concrete
// operand: the dynamic type must be the concrete type and the values
// must be equal. Form selects the value comparison.
type IfaceConcreteEqual struct {
	Iface     Expr
	Concrete  Expr
	IfaceLeft bool
	Rtti      RttiRef
	Form      string // "prim" (=== carriers) | "key" (canonical struct key)
	Negate    bool
}

func (*IfaceEqual) expr()                {}
func (e *IfaceEqual) Type() Type         { return boolType }
func (*IfaceConcreteEqual) expr()        {}
func (e *IfaceConcreteEqual) Type() Type { return boolType }

// buildIfaceEquality intercepts ==/!= where at least one operand is an
// interface (nil comparisons are handled earlier). Returns (nil, nil)
// when neither side is an interface.
func (b *builder) buildIfaceEquality(n *ast.BinaryExpr, left, right Expr, span Span) (Expr, error) {
	leftIface := left.Type().Kind == KindIface
	rightIface := right.Type().Kind == KindIface
	negate := n.Op == token.NEQ
	if !leftIface && !rightIface {
		return nil, nil
	}
	if left.Type().TypeParamName != "" || right.Type().TypeParamName != "" {
		// Type-parameter operands carry raw values, not boxes: their ==
		// stays the direct carrier comparison (the instantiation guard
		// admits only carriers whose === is exact).
		return nil, nil
	}
	if leftIface && rightIface {
		// The comparison narrows over the supertype interface's union: both
		// operands are assignable to it (Go requires == operands assignable),
		// so both boxes are its members. When the right operand is assignable
		// to the left but not vice versa, the left is the broader supertype.
		union := left.Type()
		leftGo, rightGo := b.info.Types[n.X].Type, b.info.Types[n.Y].Type
		if leftGo != nil && rightGo != nil &&
			types.AssignableTo(leftGo, rightGo) && !types.AssignableTo(rightGo, leftGo) {
			union = right.Type()
		}
		b.use("ifaceEqual")
		return &IfaceEqual{L: left, R: right, Iface: union, Negate: negate}, nil
	}
	iface, concrete := left, right
	concreteAST := n.Y
	if rightIface {
		iface, concrete = right, left
		concreteAST = n.X
	}
	concreteGoType := b.info.Types[concreteAST].Type
	rtti, err := b.rttiFor(concreteGoType, span)
	if err != nil {
		return nil, err
	}
	form := ""
	kind := concrete.Type().Kind
	switch {
	case kind.Integer() || kind.Float() || kind == KindString || kind == KindBool ||
		kind == KindPointer || kind == KindUnit:
		form = "prim"
	case kind == KindStruct && b.structEqComparable(concreteGoType):
		// The dynamic type's own generated equality (goEq$) compares.
		form = "via"
	default:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION",
			Construct: "equality between an interface and " + concrete.Type().Go, Span: span}
	}
	b.use("ifaceEqual")
	return &IfaceConcreteEqual{
		Iface: iface, Concrete: concrete, IfaceLeft: leftIface,
		Rtti: rtti, Form: form, Negate: negate,
	}, nil
}

// TupleSlot is one per-slot conversion applied to a multi-result value
// as it crosses an assignability boundary.
type TupleSlotOp int

const (
	TupleSlotPassthrough TupleSlotOp = iota
	TupleSlotBox                     // box a concrete value into an interface
	TupleSlotClone                   // copy a struct value at the boundary
)

// TupleSlot pairs one conversion with the rtti a box needs and the
// destination type the converted slot carries.
type TupleSlot struct {
	Op     TupleSlotOp
	Rtti   RttiRef
	Target Type
}

// TupleAdapt applies per-slot assignability conversions to a
// multi-result value (f(g()) forwarding, tuple returns, and tuple
// assignment): the inner value evaluates once and each slot converts.
type TupleAdapt struct {
	X       Expr
	Slots   []TupleSlot
	SrcType []Type // the inner value's result slot types (for exact typing)
	T       Type   // a synthetic tuple marker
}

func (*TupleAdapt) expr()        {}
func (a *TupleAdapt) Type() Type { return a.T }

// adaptTupleSlots wraps a multi-result value with the per-slot
// conversions each destination requires; it returns the value unchanged
// when no slot converts. sourceTuple is the inner value's Go result
// tuple; targets are the destination Go types.
func (b *builder) adaptTupleSlots(inner Expr, sourceTuple *types.Tuple, targets []types.Type, span Span) (Expr, error) {
	if sourceTuple == nil || sourceTuple.Len() != len(targets) {
		return inner, nil
	}
	slots := make([]TupleSlot, len(targets))
	any := false
	for i := range targets {
		source := sourceTuple.At(i).Type()
		target := targets[i]
		if target == nil {
			// A blank slot or an untyped destination: no conversion.
			slots[i] = TupleSlot{Op: TupleSlotPassthrough}
			continue
		}
		_, targetIsIface := target.Underlying().(*types.Interface)
		_, sourceIsIface := source.Underlying().(*types.Interface)
		switch {
		case targetIsIface && !sourceIsIface:
			// A concrete result flowing into an interface slot boxes,
			// unless it is the untyped nil (no dynamic type).
			if b.info != nil {
				// nil results cannot occur in a call's result tuple.
			}
			rtti, err := b.rttiFor(source, span)
			if err != nil {
				return nil, err
			}
			targetIR, err := b.typeOf(target, span)
			if err != nil {
				return nil, err
			}
			// A tuple slot boxes exactly like boxIfaceValue: the boxed
			// composite or external named type must join the dynamic-type
			// universe, or the empty-interface union would lack this exact
			// member and the box (and any assert-back) would be statically
			// invalid.
			if rtti.Composite != "" && rtti.ExternID == "" && !mentionsTypeParamType(source) {
				sourceIR, err := b.typeOf(source, span)
				if err != nil {
					return nil, err
				}
				mode, elem := b.compositeEqMode(source)
				b.unit.AddBoxedComposite(rtti.Composite, sourceIR, mode, elem)
			}
			b.registerBoxedExtern(source)
			slots[i] = TupleSlot{Op: TupleSlotBox, Rtti: rtti, Target: targetIR}
			any = true
		default:
			// A call's result slot is a fresh value; like a single
			// assignment (bindStructValue passes call results through),
			// only interface boxing is inserted — never a redundant copy.
			slots[i] = TupleSlot{Op: TupleSlotPassthrough}
		}
	}
	if !any {
		return inner, nil
	}
	srcTypes := make([]Type, len(targets))
	for i := range srcTypes {
		t, err := b.typeOf(sourceTuple.At(i).Type(), span)
		if err != nil {
			return nil, err
		}
		srcTypes[i] = t
	}
	b.use("tupleAdapt")
	return &TupleAdapt{X: inner, Slots: slots, SrcType: srcTypes, T: Type{Kind: KindInvalid, Go: "(tuple)"}}, nil
}
