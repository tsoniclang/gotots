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
		if _, isVar := b.info.Uses[n].(*types.Var); !isVar {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-variable " + n.Name, Span: span}
		}
		return VarTarget{Name: n.Name}, nil

	case *ast.SelectorExpr:
		selection, ok := b.info.Selections[n]
		if !ok || selection.Kind() != types.FieldVal {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "assignment to non-field selector", Span: span}
		}
		base, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if base.Type().Kind != KindPointer {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "field assignment on " + base.Type().Go, Span: span}
		}
		b.use("fieldStore")
		return &FieldTarget{X: base, Field: n.Sel.Name}, nil

	case *ast.IndexExpr:
		mapExpr, err := b.buildExpr(n.X)
		if err != nil {
			return nil, err
		}
		if mapExpr.Type().Kind != KindMap {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "indexed assignment on " + mapExpr.Type().Go, Span: span}
		}
		key, err := b.buildExpr(n.Index)
		if err != nil {
			return nil, err
		}
		b.use("mapStore")
		return &MapTarget{Map: mapExpr, Key: key}, nil
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
			definition, isNew := b.info.Defs[target].(*types.Var)
			if !isNew {
				// Go permits := to reassign existing names alongside one new
				// one; that scoping subtlety has its own reviewed lowering
				// later.
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "short declaration reusing an existing variable", Span: span}
			}
			t, err := b.typeOf(definition.Type(), span)
			if err != nil {
				return nil, err
			}
			if t.Kind == KindStruct {
				return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct value variable (value-copy semantics)", Span: span}
			}
			out.Names = append(out.Names, target.Name)
			out.Types = append(out.Types, t)
		}
		if tuple != nil {
			out.Tuple = tuple
			b.use("declTuple")
			return out, nil
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
		return out, nil

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

// buildCompoundAssign lowers x op= y into x = x op y for plain variable
// targets, whose single address evaluation is trivially preserved.
func (b *builder) buildCompoundAssign(n *ast.AssignStmt, operator token.Token) (Stmt, error) {
	span := b.span(n.Pos())
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment arity", Span: span}
	}
	target, ok := n.Lhs[0].(*ast.Ident)
	if !ok || target.Name == "_" {
		// Compound assignment to fields/indexes evaluates the target
		// address once; that shape gets its own lowering with the general
		// addressable-operand model.
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "compound assignment to non-variable", Span: span}
	}
	left, err := b.buildExpr(target)
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
		Targets: []Target{VarTarget{Name: target.Name}},
		Values:  []Expr{&Binary{Op: operator, L: left, R: right, T: operand}},
	}, nil
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
		if mapType, ok := b.info.Types[ast.Unparen(n.Rhs[0]).(*ast.IndexExpr).X]; ok {
			if goMap, isMap := mapType.Type.Underlying().(*types.Map); isMap {
				return b.typeOf(goMap.Elem(), span)
			}
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
		key, err := b.buildExpr(n.Index)
		if err != nil {
			return nil, err
		}
		b.use("mapCommaOk")
		return &MapLookup{Map: mapExpr, Key: key, T: *mapExpr.Type().Elem}, nil
	}
	return nil, nil
}
