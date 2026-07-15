package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// opAssignTokens maps compound assignment tokens to their operator.
var opAssignTokens = map[token.Token]token.Token{
	token.ADD_ASSIGN:     token.ADD,
	token.SUB_ASSIGN:     token.SUB,
	token.MUL_ASSIGN:     token.MUL,
	token.QUO_ASSIGN:     token.QUO,
	token.REM_ASSIGN:     token.REM,
	token.AND_ASSIGN:     token.AND,
	token.OR_ASSIGN:      token.OR,
	token.XOR_ASSIGN:     token.XOR,
	token.SHL_ASSIGN:     token.SHL,
	token.SHR_ASSIGN:     token.SHR,
	token.AND_NOT_ASSIGN: token.AND_NOT,
}

// buildTarget converts one assignable expression into an IR target.
func (b *builder) buildTarget(lhs ast.Expr) (Target, error) {
	span := b.span(lhs.Pos())
	switch n := ast.Unparen(lhs).(type) {
	case *ast.Ident:
		if n.Name == "_" {
			return BlankTarget{}, nil
		}
		variable, isVar := b.info.Uses[n].(*types.Var)
		if !isVar {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-variable " + n.Name, Span: span}
		}
		t, err := b.typeOf(variable.Type(), span)
		if err != nil {
			return nil, err
		}
		if _, isBoxed := b.boxedVar(n); isBoxed && boxable(t.Kind) {
			b.use("boxedStore")
			return BoxedTarget{Cell: cellName(n.Name), T: t}, nil
		}
		return VarTarget{Name: n.Name, T: t}, nil

	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[n]
		if !ok || selection.Kind() != types.FieldVal {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-field selector", Span: span}
		}
		base, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if base.Type().Kind != KindPointer && base.Type().Kind != KindStruct {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "field assignment on " + base.Type().Go, Span: span}
		}
		fieldType, err := b.typeOf(b.info.Types[lhs].Type, span)
		if err != nil {
			return nil, err
		}
		b.use("fieldStore")
		return &FieldTarget{X: base, Field: n.Sel.Name, T: fieldType}, nil

	case *ast.IndexExpr:
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		switch operand.Type().Kind {
		case KindMap:
			key, err := b.buildExprAs(n.Index, *operand.Type().Key)
			if err != nil {
				return nil, err
			}
			b.use("mapStore")
			return &MapTarget{Map: operand, Key: key}, nil
		case KindSlice:
			if operand.Type().Elem.Kind == KindExternal {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "store into a slice of external values", Span: span}
			}
			index, err := b.buildExpr(n.Index)
			if err != nil {
				return nil, err
			}
			b.use("sliceStore")
			return &SliceTarget{X: operand, Index: index}, nil
		case KindArray:
			if operand.Type().Elem.Kind == KindExternal {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "store into an array of external values", Span: span}
			}
			index, err := b.buildExpr(n.Index)
			if err != nil {
				return nil, err
			}
			b.use("arrayStore")
			return &ArrayTarget{X: operand, Index: index}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "indexed assignment on " + operand.Type().Go, Span: span}

	case *ast.StarExpr:
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind != KindPointer {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment through " + operand.Type().Go, Span: span}
		}
		b.use("pointeeStore")
		return &PointeeTarget{X: operand}, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("assignment to %T", lhs), Span: span}
}

func (b *builder) buildAssign(n *ast.AssignStmt) (Stmt, error) {
	span := b.span(n.Pos())

	if operator, isCompound := opAssignTokens[n.Tok]; isCompound {
		return b.buildCompoundAssign(n, operator)
	}

	// A single multi-valued expression spreading into all targets: a
	// multi-result call or a comma-ok map lookup.
	tuple, err := b.tupleValue(n.Rhs, len(n.Lhs))
	if err != nil {
		return nil, err
	}

	switch n.Tok {
	case token.DEFINE:
		out := &DeclStmt{}
		for i, lhs := range n.Lhs {
			target, ok := lhs.(*ast.Ident)
			if !ok {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration of non-identifier", Span: span}
			}
			if target.Name == "_" {
				// A discarded slot: the value is still evaluated. Its type
				// comes from the tuple slot or the matching right-hand side.
				t, err := b.blankSlotType(n, i, span)
				if err != nil {
					return nil, err
				}
				out.Names = append(out.Names, "_")
				out.Types = append(out.Types, t)
				continue
			}
			variable, isNew := b.info.Defs[target].(*types.Var)
			if !isNew {
				// := reassigns this existing name (Go requires at least one
				// new variable alongside it).
				existing, isVar := b.info.Uses[target].(*types.Var)
				if !isVar {
					return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration reusing a non-variable", Span: span}
				}
				variable = existing
			}
			t, err := b.typeOf(variable.Type(), span)
			if err != nil {
				return nil, err
			}
			out.Names = append(out.Names, target.Name)
			out.Types = append(out.Types, t)
			for len(out.Reused) < len(out.Names) {
				out.Reused = append(out.Reused, false)
			}
			out.Reused[len(out.Names)-1] = !isNew
		}
		anyReused := false
		for _, reused := range out.Reused {
			anyReused = anyReused || reused
		}
		if tuple != nil {
			out.Tuple = tuple
			b.use("declTuple")
			return b.boxDeclaredNames(n.Lhs, out, span)
		}
		if anyReused {
			// The non-tuple mixed form needs staged right-hand values (an
			// existing name may feed a later slot); it stays out until that
			// staging is reviewed.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration reusing an existing variable without a tuple source", Span: span}
		}
		if len(n.Rhs) != len(n.Lhs) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration arity mismatch", Span: span}
		}
		for i, rhs := range n.Rhs {
			value, err := b.buildExprAs(rhs, out.Types[i])
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		b.use("shortDecl")
		return b.boxDeclaredNames(n.Lhs, out, span)

	case token.ASSIGN:
		out := &AssignStmt{}
		for _, lhs := range n.Lhs {
			target, err := b.buildTarget(lhs)
			if err != nil {
				return nil, err
			}
			out.Targets = append(out.Targets, target)
		}
		if tuple != nil {
			out.Tuple = tuple
			b.use("assignTuple")
			return out, nil
		}
		if len(n.Rhs) != len(n.Lhs) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment arity mismatch", Span: span}
		}
		for i, rhs := range n.Rhs {
			// A blank target has no type of its own; the value is built
			// with its natural type and evaluated for effect.
			if _, isBlank := out.Targets[i].(BlankTarget); isBlank {
				value, err := b.buildExpr(rhs)
				if err != nil {
					return nil, err
				}
				out.Values = append(out.Values, value)
				continue
			}
			expected, err := b.typeOf(b.info.Types[n.Lhs[i]].Type, b.span(n.Lhs[i].Pos()))
			if err != nil {
				return nil, err
			}
			value, err := b.buildExprAs(rhs, expected)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		b.use("assign")
		return out, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment token " + n.Tok.String(), Span: span}
}

// buildCompoundAssign lowers x op= y into x = x op y. Go evaluates x's
// operands once, so the lowering — whose load and store each spell the
// operands — is restricted to shapes whose re-evaluation is pure.
func (b *builder) buildCompoundAssign(n *ast.AssignStmt, operator token.Token) (Stmt, error) {
	span := b.span(n.Pos())
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment arity", Span: span}
	}
	target, left, err := b.compoundTarget(n.Lhs[0])
	if err != nil {
		return nil, err
	}
	right, err := b.buildExpr(n.Rhs[0])
	if err != nil {
		return nil, err
	}
	operand := left.Type()
	if operand.Kind == KindString && operator != token.ADD {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + operator.String() + " on string", Span: span}
	}
	if operand.Kind == KindBool {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "compound assignment on bool", Span: span}
	}
	if operand.Kind == KindFloat32 {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "float32 arithmetic", Span: span}
	}
	b.use("compoundAssign:" + operator.String())
	return &AssignStmt{
		Targets: []Target{target},
		Values:  []Expr{&Binary{Op: operator, L: left, R: right, T: operand}},
	}, nil
}

// compoundTarget resolves the operand of a compound assignment or
// inc/dec into its store target and load expression. Go evaluates the
// operand's address once; the lowering evaluates operands in both the
// load and the store, so admitted shapes are exactly those whose
// re-evaluation is pure: variables, fields of variables, and maps or
// slices held in variables indexed by pure keys. The load runs before
// the store on both sides, preserving nil-map and bounds panic order.
func (b *builder) compoundTarget(lhs ast.Expr) (Target, Expr, error) {
	span := b.span(lhs.Pos())
	switch operand := ast.Unparen(lhs).(type) {
	case *ast.Ident:
		if operand.Name == "_" {
			return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment to the blank identifier", Span: span}
		}
		load, err := b.buildExpr(operand)
		if err != nil {
			return nil, nil, err
		}
		if _, isBoxed := b.boxedVar(operand); isBoxed && boxable(load.Type().Kind) {
			b.use("boxedStore")
			return BoxedTarget{Cell: cellName(operand.Name), T: load.Type()}, load, nil
		}
		return VarTarget{Name: operand.Name, T: load.Type()}, load, nil

	case *ast.SelectorExpr:
		if !b.pureFieldBase(operand.X) {
			return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment to a field with an impure base", Span: span}
		}
		built, err := b.buildExpr(operand)
		if err != nil {
			return nil, nil, err
		}
		load, isField := built.(*FieldLoad)
		if !isField {
			return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment to a non-field selector", Span: span}
		}
		b.use("fieldStore")
		return &FieldTarget{X: load.X, Field: load.Field, T: load.T}, load, nil

	case *ast.IndexExpr:
		if _, baseIsIdent := ast.Unparen(operand.X).(*ast.Ident); !baseIsIdent {
			return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "indexed compound assignment on a non-variable operand", Span: span}
		}
		if !b.pureOperand(operand.Index) {
			return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "indexed compound assignment with a non-pure index", Span: span}
		}
		built, err := b.buildExpr(operand)
		if err != nil {
			return nil, nil, err
		}
		switch load := built.(type) {
		case *MapGet:
			b.use("mapStore")
			return &MapTarget{Map: load.Map, Key: load.Key}, load, nil
		case *SliceGet:
			b.use("sliceStore")
			return &SliceTarget{X: load.X, Index: load.Index}, load, nil
		case *ArrayGet:
			b.use("arrayStore")
			return &ArrayTarget{X: load.X, Index: load.Index}, load, nil
		}
		return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "indexed compound assignment on " + built.Type().Go, Span: span}
	}
	return nil, nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("compound assignment to %T", lhs), Span: span}
}

// pureFieldBase reports whether re-evaluating a compound target's base
// is exact: chains of variables, field selections, and dereferences —
// no calls or indexing. A nil-pointer panic surfaces on the load
// re-evaluation first, so re-running the chain observes nothing new.
func (b *builder) pureFieldBase(e ast.Expr) bool {
	switch base := ast.Unparen(e).(type) {
	case *ast.Ident:
		return true
	case *ast.StarExpr:
		return b.pureFieldBase(base.X)
	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[base]
		if !ok || selection.Kind() != types.FieldVal {
			return false
		}
		return b.pureFieldBase(base.X)
	}
	return false
}

// pureOperand reports whether re-evaluating the expression is exact: a
// folded constant or a plain variable reference.
func (b *builder) pureOperand(e ast.Expr) bool {
	if tv, ok := b.info.Types[e]; ok && tv.Value != nil {
		return true
	}
	_, isIdent := ast.Unparen(e).(*ast.Ident)
	return isIdent
}

// blankSlotType resolves the type of a discarded (_) slot from the tuple
// result or the matching right-hand expression.
func (b *builder) blankSlotType(n *ast.AssignStmt, index int, span Span) (Type, error) {
	if len(n.Rhs) == 1 && len(n.Lhs) > 1 {
		if tuple, ok := b.info.Types[ast.Unparen(n.Rhs[0])].Type.(*types.Tuple); ok && tuple.Len() == len(n.Lhs) {
			return b.typeOf(tuple.At(index).Type(), span)
		}
		// Comma-ok forms: value type for slot 0, bool for slot 1.
		if index == 1 {
			return Type{Kind: KindBool, Go: "bool"}, nil
		}
		switch rhs := ast.Unparen(n.Rhs[0]).(type) {
		case *ast.IndexExpr:
			if mapType, ok := b.info.Types[rhs.X]; ok {
				if goMap, isMap := mapType.Type.Underlying().(*types.Map); isMap {
					return b.typeOf(goMap.Elem(), span)
				}
			}
		case *ast.TypeAssertExpr:
			return b.typeOf(b.info.Types[rhs.Type].Type, span)
		}
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "blank slot in unrecognized tuple form", Span: span}
	}
	if index < len(n.Rhs) {
		return b.typeOf(b.info.Types[n.Rhs[index]].Type, span)
	}
	return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "blank slot arity", Span: span}
}

// tupleValue recognizes a single expression whose multiple values
// initialize several targets: a multi-result call or a comma-ok map
// lookup. It returns nil when the shape does not apply.
func (b *builder) tupleValue(rhs []ast.Expr, targetCount int) (Expr, error) {
	if len(rhs) != 1 || targetCount < 2 {
		return nil, nil
	}
	switch n := ast.Unparen(rhs[0]).(type) {
	case *ast.CallExpr:
		if _, isConversion := b.conversionTarget(n); isConversion {
			return nil, nil
		}
		tuple, ok := b.info.Types[n].Type.(*types.Tuple)
		if !ok || tuple.Len() != targetCount {
			return nil, nil
		}
		return b.buildAnyCall(n)
	case *ast.IndexExpr:
		if targetCount != 2 {
			return nil, nil
		}
		mapExpr, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if mapExpr.Type().Kind != KindMap {
			return nil, nil
		}
		key, err := b.buildExprAs(n.Index, *mapExpr.Type().Key)
		if err != nil {
			return nil, err
		}
		b.use("mapCommaOk")
		return &MapLookup{Map: mapExpr, Key: key, T: *mapExpr.Type().Elem}, nil
	case *ast.TypeAssertExpr:
		if targetCount != 2 || n.Type == nil {
			return nil, nil
		}
		return b.buildTypeAssert(n, true)
	}
	return nil, nil
}
