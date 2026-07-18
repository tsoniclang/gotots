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
			return nil, &Unsupported{Kind: KindAssignmentToNonVariable, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-variable " + n.Name, Span: span}
		}
		t, err := b.typeOf(variable.Type(), span)
		if err != nil {
			return nil, err
		}
		if _, isBoxed := b.boxedVar(n); isBoxed && boxable(t.Kind) {
			b.use("boxedStore")
			return BoxedTarget{Cell: cellName(n.Name), T: t}, nil
		}
		pkg := ""
		if variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
			pkg = variable.Pkg().Path()
		}
		return VarTarget{Name: n.Name, Pkg: pkg, T: t}, nil

	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[n]
		if !ok || selection.Kind() != types.FieldVal {
			return nil, &Unsupported{Kind: KindAssignmentToNonFieldSelector, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-field selector", Span: span}
		}
		base, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		// (*p).f assigns through p exactly like p.f: keep the POINTER as the
		// base so the nil dereference panics at the STORE (Go's two-phase
		// assignment evaluates the right-hand side first, then carries out
		// the store) — never eagerly at base evaluation, which would skip a
		// side-effecting right-hand side.
		if deref, ok := base.(*Deref); ok && deref.X.Type().Kind == KindPointer {
			base = deref.X
		}
		if path := selection.Index(); len(path) > 1 {
			// A promoted field target: the base chains through the
			// embedded fields; the final segment is the stored field.
			base, err = b.chainFieldPath(base, b.info.Types[n.X].Type, path[:len(path)-1], span)
			if err != nil {
				return nil, err
			}
		}
		if base.Type().Kind != KindPointer && base.Type().Kind != KindStruct {
			return nil, &Unsupported{Kind: KindFieldAssignmentOn, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "field assignment on " + base.Type().Go, Span: span}
		}
		fieldType, err := b.typeOf(b.info.Types[lhs].Type, span)
		if err != nil {
			return nil, err
		}
		b.use("fieldStore")
		cell, err := b.fieldCell(selection, fieldType)
		if err != nil {
			return nil, err
		}
		return &FieldTarget{X: base, Field: n.Sel.Name, T: fieldType, Cell: cell}, nil

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
				return nil, &Unsupported{Kind: KindStoreIntoASliceOfExternalValues, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "store into a slice of external values", Span: span}
			}
			index, err := b.buildExpr(n.Index)
			if err != nil {
				return nil, err
			}
			b.use("sliceStore")
			return &SliceTarget{X: operand, Index: index}, nil
		case KindArray:
			if operand.Type().Elem.Kind == KindExternal {
				return nil, &Unsupported{Kind: KindStoreIntoAnArrayOfExternalValues, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "store into an array of external values", Span: span}
			}
			index, err := b.buildExpr(n.Index)
			if err != nil {
				return nil, err
			}
			b.use("arrayStore")
			return &ArrayTarget{X: operand, Index: index}, nil
		}
		return nil, &Unsupported{Kind: KindIndexedAssignmentOn, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "indexed assignment on " + operand.Type().Go, Span: span}

	case *ast.StarExpr:
		operand, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind != KindPointer {
			return nil, &Unsupported{Kind: KindAssignmentThrough, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment through " + operand.Type().Go, Span: span}
		}
		b.use("pointeeStore")
		return &PointeeTarget{X: operand}, nil
	}
	return nil, &Unsupported{Kind: KindAssignmentTo, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: fmt.Sprintf("assignment to %T", lhs), Span: span}
}

func (b *builder) buildAssign(n *ast.AssignStmt) (Stmt, error) {
	span := b.span(n.Pos())

	if operator, isCompound := opAssignTokens[n.Tok]; isCompound {
		return b.buildCompoundAssign(n, operator)
	}

	// A single multi-valued expression spreading into all targets: a
	// multi-result call or a comma-ok map lookup.
	targets := make([]types.Type, len(n.Lhs))
	for i, lhs := range n.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
			if def := b.info.Defs[ident]; def != nil {
				targets[i] = def.Type()
			} else if tv, ok := b.info.Types[lhs]; ok {
				targets[i] = tv.Type
			}
		} else if tv, ok := b.info.Types[lhs]; ok {
			targets[i] = tv.Type
		}
	}
	tuple, err := b.tupleValue(n.Rhs, len(n.Lhs), targets)
	if err != nil {
		return nil, err
	}

	switch n.Tok {
	case token.DEFINE:
		out := &DeclStmt{}
		for i, lhs := range n.Lhs {
			target, ok := lhs.(*ast.Ident)
			if !ok {
				return nil, &Unsupported{Kind: KindShortDeclarationOfNonIdentifier, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration of non-identifier", Span: span}
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
					return nil, &Unsupported{Kind: KindShortDeclarationReusingANonVariable, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration reusing a non-variable", Span: span}
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
			return nil, &Unsupported{Kind: KindShortDeclarationReusingAnExistingVariableWithoutATupleSource, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration reusing an existing variable without a tuple source", Span: span}
		}
		if len(n.Rhs) != len(n.Lhs) {
			return nil, &Unsupported{Kind: KindShortDeclarationArityMismatch, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration arity mismatch", Span: span}
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
			return nil, &Unsupported{Kind: KindAssignmentArityMismatch, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment arity mismatch", Span: span}
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
	return nil, &Unsupported{Kind: KindAssignmentToken, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment token " + n.Tok.String(), Span: span}
}

// buildCompoundAssign lowers x op= y: the target's operands stage
// exactly once at emission, so every assignable shape is admissible.
func (b *builder) buildCompoundAssign(n *ast.AssignStmt, operator token.Token) (Stmt, error) {
	span := b.span(n.Pos())
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
		return nil, &Unsupported{Kind: KindCompoundAssignmentArity, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment arity", Span: span}
	}
	operandT, err := b.typeOf(b.info.Types[n.Lhs[0]].Type, span)
	if err != nil {
		return nil, err
	}
	if err := compoundOpAdmissible(operator, operandT, span); err != nil {
		return nil, err
	}
	target, err := b.buildTarget(n.Lhs[0])
	if err != nil {
		return nil, err
	}
	if _, isBlank := target.(BlankTarget); isBlank {
		return nil, &Unsupported{Kind: KindCompoundAssignmentToTheBlankIdentifier, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment to the blank identifier", Span: span}
	}
	right, err := b.buildExprAs(n.Rhs[0], operandT)
	if err != nil {
		return nil, err
	}
	b.use("compoundAssign:" + operator.String())
	return &CompoundStmt{Target: target, Op: operator, Rhs: right, OperandT: operandT}, nil
}

// compoundOpAdmissible mirrors the binary-operation review for the
// compound forms.
func compoundOpAdmissible(operator token.Token, operand Type, span Span) error {
	if operand.Kind == KindString {
		if operator != token.ADD {
			return &Unsupported{Kind: KindOperator, Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + operator.String() + " on string", Span: span}
		}
		return nil
	}
	if operand.Kind == KindFloat32 {
		return &Unsupported{Kind: KindFloat32Arithmetic, Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "float32 arithmetic", Span: span}
	}
	if !operand.Kind.Integer() && !operand.Kind.Float() {
		return &Unsupported{Kind: KindCompoundAssignmentOn, Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "compound assignment on " + operand.Go, Span: span}
	}
	if operand.Kind.Float() && operator != token.ADD && operator != token.SUB &&
		operator != token.MUL && operator != token.QUO {
		return &Unsupported{Kind: KindOperator, Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + operator.String() + " on " + operand.Go, Span: span}
	}
	return nil
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
		return Type{}, &Unsupported{Kind: KindBlankSlotInUnrecognizedTupleForm, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "blank slot in unrecognized tuple form", Span: span}
	}
	if index < len(n.Rhs) {
		return b.typeOf(b.info.Types[n.Rhs[index]].Type, span)
	}
	return Type{}, &Unsupported{Kind: KindBlankSlotArity, Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "blank slot arity", Span: span}
}

// tupleValue recognizes a single expression whose multiple values
// initialize several targets: a multi-result call or a comma-ok map
// lookup. It returns nil when the shape does not apply.
func (b *builder) tupleValue(rhs []ast.Expr, targetCount int, targets []types.Type) (Expr, error) {
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
		call, err := b.buildAnyCall(n)
		if err != nil {
			return nil, err
		}
		// Each destination slot undergoes the same assignability
		// conversion an ordinary single assignment would.
		return b.adaptTupleSlots(call, tuple, targets, b.span(n.Pos()))
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
		lookup := Expr(&MapLookup{Map: mapExpr, Key: key, T: *mapExpr.Type().Elem})
		// The value slot boxes when the destination is an interface; the
		// ok slot is always bool.
		if len(targets) == 2 && targets[0] != nil {
			mapType := types.Unalias(b.info.Types[n.X].Type).Underlying().(*types.Map)
			sourceTuple := types.NewTuple(
				types.NewVar(0, nil, "", mapType.Elem()),
				types.NewVar(0, nil, "", types.Typ[types.Bool]))
			return b.adaptTupleSlots(lookup, sourceTuple, targets, b.span(n.Pos()))
		}
		return lookup, nil
	case *ast.TypeAssertExpr:
		if targetCount != 2 || n.Type == nil {
			return nil, nil
		}
		return b.buildTypeAssert(n, true)
	}
	return nil, nil
}
