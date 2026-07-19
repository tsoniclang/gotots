// Struct-literal construction and value-copy binding.
package ir

import (
	"go/ast"
	"go/types"
)

// bindStructValue inserts the value copy of a struct binding. Loads from
// variables, fields, and elements clone; already-fresh values — new
// allocations, zeros, copies, and call results (copied at their return
// sites) — bind directly.
func (b *builder) bindStructValue(e Expr) Expr {
	kind := e.Type().Kind
	if kind != KindStruct && kind != KindArray && kind != KindExternal {
		return e
	}
	switch e.(type) {
	case *StructNew, *StructZero, *StructCopy, *Call, *MethodCall, *ArrayLit, *ExternZero, *ExternalMethodCall, *ExternalCall:
		return e
	}
	switch kind {
	case KindArray:
		b.use("arrayCopy")
	case KindExternal:
		b.use("externCopy")
	default:
		b.use("structCopy")
	}
	return &StructCopy{X: e}
}

// buildStructLit materializes a struct composite literal — as a heap
// allocation (&T{...}, pointer type) or a fresh struct value (T{...}) —
// with every field value in declaration order (omitted fields are
// explicit zeros).
func (b *builder) buildStructLit(lit *ast.CompositeLit, t Type) (Expr, error) {
	span := b.span(lit.Pos())
	litGoType := b.info.Types[lit].Type
	if pointer, isPointer := types.Unalias(litGoType).(*types.Pointer); isPointer {
		// The elided allocation form: the literal's type is the pointee.
		litGoType = pointer.Elem()
	}
	structType, isAnon := types.Unalias(litGoType).(*types.Struct)
	if !isAnon {
		structType = types.Unalias(litGoType).(*types.Named).Underlying().(*types.Struct)
	}

	fieldIRType := func(field *types.Var) (Type, error) { return b.typeOf(field.Type(), span) }
	fieldByName := map[string]*types.Var{}
	for i := range structType.NumFields() {
		fieldByName[structType.Field(i).Name()] = structType.Field(i)
	}

	// Resolve provided values, keyed or positional, typing each against
	// its field so nil literals get exact zeros. Source order is
	// recorded: Go evaluates literal values in the order written.
	provided := map[string]Expr{}
	var sourceOrder []string
	if len(lit.Elts) > 0 {
		_, keyed := lit.Elts[0].(*ast.KeyValueExpr)
		for index, element := range lit.Elts {
			if keyValue, isKeyed := element.(*ast.KeyValueExpr); isKeyed != keyed {
				return nil, &Unsupported{Kind: KindMixedKeyedAndPositionalLiteral, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "mixed keyed and positional literal", Span: span}
			} else if isKeyed {
				name := keyValue.Key.(*ast.Ident).Name
				expected, err := fieldIRType(fieldByName[name])
				if err != nil {
					return nil, err
				}
				value, err := b.buildExprAs(keyValue.Value, expected)
				if err != nil {
					return nil, err
				}
				provided[name] = value
				sourceOrder = append(sourceOrder, name)
			} else {
				expected, err := fieldIRType(structType.Field(index))
				if err != nil {
					return nil, err
				}
				value, err := b.buildExprAs(element, expected)
				if err != nil {
					return nil, err
				}
				provided[structType.Field(index).Name()] = value
				sourceOrder = append(sourceOrder, structType.Field(index).Name())
			}
		}
	}

	out := &StructNew{Pkg: t.Pkg, TypeName: t.Named, T: t}
	argIndexOf := map[string]int{}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		if field.Name() == "_" && zeroSizeType(field.Type()) {
			// A no-output zero-size blank field is absent from the generated
			// class; construction skips it too (a literal can never name it).
			continue
		}
		if value, ok := provided[field.Name()]; ok {
			argIndexOf[field.Name()] = len(out.Args)
			out.Args = append(out.Args, value)
			continue
		}
		fieldType, err := b.typeOf(field.Type(), span)
		if err != nil {
			return nil, err
		}
		zero, err := zeroValue(fieldType, span)
		if err != nil {
			return nil, err
		}
		out.Args = append(out.Args, zero)
	}
	// A keyed literal whose source order differs from field order stages
	// its provided values so they evaluate in source order.
	ascending := true
	order := make([]int, 0, len(sourceOrder))
	for _, name := range sourceOrder {
		order = append(order, argIndexOf[name])
	}
	for k := 1; k < len(order); k++ {
		if order[k] < order[k-1] {
			ascending = false
			break
		}
	}
	if !ascending {
		out.EvalOrder = order
		b.use("structLitStaged")
	}
	b.use("structNew")
	return out, nil
}

// constValue renders an exact go/constant value for the resolved type.
