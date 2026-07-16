package ir

import (
	"go/ast"

	"go/types"
)

// builtinCallee resolves a call to a predeclared operation by object
// identity, never spelling.
func (b *builder) builtinCallee(call *ast.CallExpr) (*types.Builtin, bool) {
	ident, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok {
		return nil, false
	}
	builtin, ok := b.info.Uses[ident].(*types.Builtin)
	return builtin, ok
}

// buildBuiltin lowers the reviewed predeclared operations.
func (b *builder) buildBuiltin(call *ast.CallExpr, builtin *types.Builtin, resultType types.Type) (Expr, error) {
	span := b.span(call.Pos())
	switch builtin.Name() {
	case "len":
		operand, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		switch operand.Type().Kind {
		case KindString:
			b.use("len:string")
			return &StringLen{X: operand}, nil
		case KindMap:
			b.use("len:map")
			return &MapLen{X: operand}, nil
		case KindSlice:
			b.use("len:slice")
			return &SliceLen{X: operand}, nil
		case KindArray:
			b.use("len:array")
			return &ArrayLenExpr{X: operand}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "len of " + operand.Type().Go, Span: span}
	case "cap":
		operand, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind == KindSlice {
			b.use("cap:slice")
			return &SliceCap{X: operand}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "cap of " + operand.Type().Go, Span: span}
	case "copy":
		dst, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		src, err := b.buildExpr(call.Args[1])
		if err != nil {
			return nil, err
		}
		if dst.Type().Kind != KindSlice || src.Type().Kind != KindSlice {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "copy between " + dst.Type().Go + " and " + src.Type().Go, Span: span}
		}
		if dst.Type().Elem.Kind == KindArray {
			// copy() overwrites each destination array's memory; the ABI
			// struct copy path requires goSet$ carriers.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "copy of fixed-array elements", Span: span}
		}
		b.use("copy:slice")
		return &SliceCopy{Dst: dst, Src: src}, nil
	case "append":
		operand, err := b.buildExpr(call.Args[0])
		if err != nil {
			return nil, err
		}
		if operand.Type().Kind != KindSlice {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "append to " + operand.Type().Go, Span: span}
		}
		if operand.Type().Elem.Kind == KindArray {
			// Appended values must copy into the backing store; the ABI
			// struct append path requires goClone$/goSet$ carriers.
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "append of fixed-array elements", Span: span}
		}
		if call.Ellipsis.IsValid() {
			source, err := b.buildExprAs(call.Args[1], operand.Type())
			if err != nil {
				return nil, err
			}
			b.use("append:spread")
			return &SliceAppendSlice{X: operand, Source: source, T: operand.Type()}, nil
		}
		out := &SliceAppend{X: operand, T: operand.Type()}
		for _, arg := range call.Args[1:] {
			value, err := b.buildExprAs(arg, *operand.Type().Elem)
			if err != nil {
				return nil, err
			}
			out.Values = append(out.Values, value)
		}
		b.use("append")
		return out, nil
	case "make":
		t, err := b.typeOf(resultType, span)
		if err != nil {
			return nil, err
		}
		if t.Kind == KindMap && len(call.Args) == 1 {
			b.use("makeMap")
			return &MapMake{T: t}, nil
		}
		if t.Kind == KindMap && len(call.Args) == 2 {
			// The capacity hint changes no observable behavior (Go
			// tolerates any int hint) but still evaluates.
			hint, err := b.buildExpr(call.Args[1])
			if err != nil {
				return nil, err
			}
			b.use("makeMap")
			return &MapMake{T: t, Hint: hint}, nil
		}
		if t.Kind == KindSlice && (len(call.Args) == 2 || len(call.Args) == 3) {
			length, err := b.buildExpr(call.Args[1])
			if err != nil {
				return nil, err
			}
			out := &SliceMake{T: t, Length: length}
			if len(call.Args) == 3 {
				capacity, err := b.buildExpr(call.Args[2])
				if err != nil {
					return nil, err
				}
				out.Capacity = capacity
			}
			b.use("makeSlice")
			return out, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "make of " + t.Go, Span: span}
	}
	switch builtin.Name() {
	case "min", "max":
		t, err := b.typeOf(resultType, span)
		if err != nil {
			return nil, err
		}
		if !t.Kind.Integer() && !t.Kind.Float() && t.Kind != KindString {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "builtin " + builtin.Name() + " over " + t.Go, Span: span}
		}
		if t.Kind == KindFloat32 {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "float32 arithmetic", Span: span}
		}
		out := &MinMax{Max: builtin.Name() == "max", T: t}
		for _, arg := range call.Args {
			built, err := b.buildExprAs(arg, t)
			if err != nil {
				return nil, err
			}
			out.Args = append(out.Args, built)
		}
		b.use(builtin.Name())
		return out, nil
	case "new":
		t, err := b.typeOf(resultType, span)
		if err != nil {
			return nil, err
		}
		if t.Kind != KindPointer || t.Elem == nil {
			return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "new of " + t.Go, Span: span}
		}
		if t.Elem.Kind == KindStruct {
			// new(T) for a named struct: the fresh zero instance is the
			// pointer (object identity is the address).
			b.use("new:struct")
			return &AddrOf{X: &StructZero{T: *t.Elem}, T: t}, nil
		}
		if boxable(t.Elem.Kind) {
			zero, err := zeroValue(*t.Elem, span)
			if err != nil {
				return nil, err
			}
			b.use("new:cell")
			return &CellNew{Zero: zero, T: t}, nil
		}
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "new of " + t.Go, Span: span}
	}
	return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_OPERATION", Construct: "builtin " + builtin.Name(), Span: span}
}

// MinMax is the min/max builtin over ordered carriers: arguments
// evaluate left to right; float NaN propagates and -0/+0 order exactly
// as Go defines; strings compare byte-wise.
type MinMax struct {
	Max  bool
	Args []Expr
	T    Type
}

func (*MinMax) expr()        {}
func (m *MinMax) Type() Type { return m.T }

func (b *builder) buildMapDelete(call *ast.CallExpr) (Stmt, error) {
	mapExpr, err := b.buildExpr(call.Args[0])
	if err != nil {
		return nil, err
	}
	key, err := b.buildExpr(call.Args[1])
	if err != nil {
		return nil, err
	}
	b.use("mapDelete")
	return &MapDeleteStmt{Map: mapExpr, Key: key}, nil
}

func (b *builder) buildClear(call *ast.CallExpr) (Stmt, error) {
	operand, err := b.buildExpr(call.Args[0])
	if err != nil {
		return nil, err
	}
	if operand.Type().Kind != KindMap {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "clear of " + operand.Type().Go, Span: b.span(call.Pos())}
	}
	b.use("mapClear")
	return &MapClearStmt{Map: operand}, nil
}

// buildPanic lowers panic(v) for argument kinds whose Go %v formatting
// coincides with JS String(): strings, canonical-range and bigint
// integers, and booleans.
func (b *builder) buildPanic(call *ast.CallExpr) (Stmt, error) {
	span := b.span(call.Pos())
	value, err := b.buildExpr(call.Args[0])
	if err != nil {
		return nil, err
	}
	kind := value.Type().Kind
	errorType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	if goType := b.info.Types[call.Args[0]].Type; goType != nil && types.Implements(goType, errorType) {
		// panic(err) for any error-implementing type (interface or a
		// concrete value): the panic retains the typed value and formats
		// through its dynamic Error method — an error box carries the
		// dynamic type and payload.
		boxed := value
		if kind != KindIface {
			errType, err := b.typeOf(types.Universe.Lookup("error").Type(), span)
			if err != nil {
				return nil, err
			}
			boxed, err = b.boxIfaceValue(value, goType, errType, span)
			if err != nil {
				return nil, err
			}
		}
		b.use("panic:error")
		return &PanicStmt{Value: boxed, IsError: true, ErrorKey: MethodKey(errorType.Method(0))}, nil
	}
	if kind != KindString && kind != KindBool && !kind.Integer() {
		return nil, &Unsupported{Code: "GOTOTS_UNSUPPORTED_STATEMENT", Construct: "panic with " + value.Type().Go + " (formatting not reviewed)", Span: span}
	}
	b.use("panic")
	return &PanicStmt{Value: value}, nil
}
