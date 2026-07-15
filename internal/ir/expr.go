package ir

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math/big"
	"strconv"
)

// buildExpr converts one typed Go expression into IR, folding constants
// through go/constant so no textual constant parsing ever occurs.
func (b *builder) buildExpr(e ast.Expr) (Expr, error) {
	span := b.span(e.Pos())
	tv, ok := b.info.Types[e]
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "expression without type evidence", Span: span}
	}

	// Exact constant folding: any expression the Go type checker evaluated
	// to a constant becomes a Const with the exact value.
	if tv.Value != nil {
		t, err := typeOf(tv.Type, span)
		if err != nil {
			return nil, err
		}
		value, err := constValue(tv.Value, t, span)
		if err != nil {
			return nil, err
		}
		b.use("const")
		return &Const{T: t, Value: value}, nil
	}

	switch n := e.(type) {
	case *ast.ParenExpr:
		return b.buildExpr(n.X)

	case *ast.Ident:
		object := b.info.Uses[n]
		variable, ok := object.(*types.Var)
		if !ok {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("identifier %q (%T)", n.Name, object), Span: span}
		}
		t, err := typeOf(variable.Type(), span)
		if err != nil {
			return nil, err
		}
		b.use("varRef")
		return &VarRef{Name: n.Name, T: t}, nil

	case *ast.BinaryExpr:
		return b.buildBinary(n, tv.Type)

	case *ast.UnaryExpr:
		switch n.Op {
		case token.SUB, token.NOT, token.XOR, token.ADD:
			x, err := b.buildExpr(n.X)
			if err != nil {
				return nil, err
			}
			if n.Op == token.ADD {
				return x, nil // unary plus is identity
			}
			t, err := typeOf(tv.Type, span)
			if err != nil {
				return nil, err
			}
			b.use("unary:" + n.Op.String())
			return &Unary{Op: n.Op, X: x, T: t}, nil
		default:
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "unary operator " + n.Op.String(), Span: span}
		}

	case *ast.CallExpr:
		if tv.IsType() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "type in call position", Span: span}
		}
		if convert, is := b.conversionTarget(n); is {
			x, err := b.buildExpr(n.Args[0])
			if err != nil {
				return nil, err
			}
			to, err := typeOf(convert, span)
			if err != nil {
				return nil, err
			}
			if err := b.checkConversion(x.Type(), to, span); err != nil {
				return nil, err
			}
			b.use("convert")
			return &Convert{X: x, To: to}, nil
		}
		call, err := b.buildCall(n)
		if err != nil {
			return nil, err
		}
		if len(call.Results) != 1 {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call with %d results in expression position", len(call.Results)), Span: span}
		}
		return call, nil
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("%T", e), Span: span}
}

func (b *builder) buildBinary(n *ast.BinaryExpr, resultType types.Type) (Expr, error) {
	span := b.span(n.Pos())
	left, err := b.buildExpr(n.X)
	if err != nil {
		return nil, err
	}
	right, err := b.buildExpr(n.Y)
	if err != nil {
		return nil, err
	}
	t, err := typeOf(resultType, span)
	if err != nil {
		return nil, err
	}
	operand := left.Type()

	switch n.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
		token.AND, token.OR, token.XOR, token.AND_NOT,
		token.SHL, token.SHR:
		if n.Op == token.ADD && operand.Kind == KindString {
			// String concatenation is exact under direct JS +.
		} else if !operand.Kind.Integer() && !operand.Kind.Float() {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + n.Op.String() + " on " + operand.Go, Span: span}
		}
		if operand.Kind.Float() && (n.Op == token.REM || n.Op == token.AND || n.Op == token.OR ||
			n.Op == token.XOR || n.Op == token.AND_NOT || n.Op == token.SHL || n.Op == token.SHR) {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + n.Op.String() + " on " + operand.Go, Span: span}
		}
		if operand.Kind == KindFloat32 {
			// Exact float32 rounding points are not lowered yet.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "float32 arithmetic", Span: span}
		}
	case token.EQL, token.NEQ:
		// Equality is exact for the whole subset (UTF-8 string equality
		// coincides with JS code-unit equality).
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		if operand.Kind == KindString {
			// Go compares UTF-8 bytes; JS compares UTF-16 code units. The
			// orders differ outside ASCII, so string ordering needs its own
			// reviewed lowering.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "string ordering comparison", Span: span}
		}
	case token.LAND, token.LOR:
		// Short-circuit boolean semantics match JS exactly.
	default:
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "operator " + n.Op.String(), Span: span}
	}
	b.use("binary:" + n.Op.String())
	return &Binary{Op: n.Op, L: left, R: right, T: t}, nil
}

// conversionTarget reports whether the call is a type conversion and, if
// so, the target type — decided by type-checker evidence, never spelling.
func (b *builder) conversionTarget(call *ast.CallExpr) (types.Type, bool) {
	if tv, ok := b.info.Types[call.Fun]; ok && tv.IsType() && len(call.Args) == 1 {
		return tv.Type, true
	}
	return nil, false
}

// checkConversion admits only conversions in the reviewed subset:
// integer-to-integer of any widths, and integer-to-float64.
func (b *builder) checkConversion(from, to Type, span Span) error {
	if from.Kind.Integer() && to.Kind.Integer() {
		return nil
	}
	if from.Kind.Integer() && to.Kind == KindFloat64 {
		return nil
	}
	return &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION",
		Construct: "conversion from " + from.Go + " to " + to.Go, Span: span}
}

func (b *builder) buildCall(n *ast.CallExpr) (*Call, error) {
	span := b.span(n.Pos())
	callee, ok := ast.Unparen(n.Fun).(*ast.Ident)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", n.Fun), Span: span}
	}
	object := b.info.Uses[callee]
	function, ok := object.(*types.Func)
	if !ok {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: fmt.Sprintf("call of %T", object), Span: span}
	}
	if function.Pkg() == nil || function.Pkg().Path() != b.pkgPath {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "call outside the translated package", Span: span}
	}
	signature := function.Type().(*types.Signature)
	if signature.Recv() != nil || signature.Variadic() || signature.TypeParams() != nil {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "method, variadic, or generic call", Span: span}
	}
	if n.Ellipsis.IsValid() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "slice-expansion call", Span: span}
	}

	call := &Call{Callee: function.Name()}
	for _, arg := range n.Args {
		built, err := b.buildExpr(arg)
		if err != nil {
			return nil, err
		}
		call.Args = append(call.Args, built)
	}
	results := signature.Results()
	for i := range results.Len() {
		t, err := typeOf(results.At(i).Type(), span)
		if err != nil {
			return nil, err
		}
		call.Results = append(call.Results, t)
	}
	b.use("call")
	return call, nil
}

// constValue renders an exact go/constant value for the resolved type.
func constValue(v constant.Value, t Type, span Span) (string, error) {
	switch t.Kind {
	case KindBool:
		return fmt.Sprintf("%v", constant.BoolVal(v)), nil
	case KindString:
		quoted, err := json.Marshal(constant.StringVal(v))
		if err != nil {
			return "", err
		}
		return string(quoted), nil
	case KindFloat32, KindFloat64:
		f, _ := constant.Float64Val(v)
		return strconv.FormatFloat(f, 'g', -1, 64), nil
	default:
		if !t.Kind.Integer() {
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "constant of type " + t.Go, Span: span}
		}
		text := v.ExactString()
		if _, ok := new(big.Int).SetString(text, 10); !ok {
			return "", &Unsupported{Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-integral integer constant " + text, Span: span}
		}
		return text, nil
	}
}
