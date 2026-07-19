// External-contract expression bridges: keyed/positional composite
// literals, exported-field reads, and owned↔external identical-underlying
// conversions — each a typed fail-closed stub obligation the emulation
// layer implements.
package ir

import (
	"go/ast"
	"go/types"
)

// buildExternLit lowers a KEYED composite literal of an external struct
// type to its reviewed constructor stub: one typed obligation per
// distinct field set, implemented by the emulation layer (fail-closed
// until then). Positional or non-identifier keys stay outside the
// reviewed surface.
func (b *builder) buildExternLit(n *ast.CompositeLit, t Type, goType types.Type, span Span) (Expr, error) {
	structType, isStruct := types.Unalias(goType).Underlying().(*types.Struct)
	if !isStruct {
		return nil, &Unsupported{Kind: KindCompositeLiteralOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "composite literal of " + t.Go, Span: span}
	}
	fieldTypeOf := map[string]types.Type{}
	for i := range structType.NumFields() {
		fieldTypeOf[structType.Field(i).Name()] = structType.Field(i).Type()
	}
	fields := make([]string, 0, len(n.Elts))
	values := make([]Expr, 0, len(n.Elts))
	var fieldTypes []Type
	if len(n.Elts) > 0 {
		if _, isKV := n.Elts[0].(*ast.KeyValueExpr); !isKV {
			// POSITIONAL form: Go requires every field, in declaration
			// order — the constructor obligation covers the full field
			// set.
			if len(n.Elts) != structType.NumFields() {
				return nil, &Unsupported{Kind: KindCompositeLiteralOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "partial positional composite literal of external " + t.Go, Span: span}
			}
			for i, element := range n.Elts {
				field := structType.Field(i)
				fieldType, err := b.typeOf(field.Type(), span)
				if err != nil {
					return nil, err
				}
				value, err := b.buildExprAs(element, fieldType)
				if err != nil {
					return nil, err
				}
				fields = append(fields, field.Name())
				values = append(values, value)
				fieldTypes = append(fieldTypes, fieldType)
			}
			obligation := b.unit.AddExternalType(t.Pkg, t.Named)
			symbol := obligation.AddLiteralShape(fields, fieldTypes)
			b.use("externLit")
			return &ExternLit{T: t, Symbol: symbol, Values: values}, nil
		}
	}
	for _, element := range n.Elts {
		kv, isKV := element.(*ast.KeyValueExpr)
		if !isKV {
			return nil, &Unsupported{Kind: KindCompositeLiteralOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "mixed composite literal of external " + t.Go, Span: span}
		}
		key, isIdent := kv.Key.(*ast.Ident)
		if !isIdent {
			return nil, &Unsupported{Kind: KindCompositeLiteralOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-identifier key in composite literal of external " + t.Go, Span: span}
		}
		fieldGoType, has := fieldTypeOf[key.Name]
		if !has {
			return nil, &Unsupported{Kind: KindCompositeLiteralOf, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "unknown field " + key.Name + " in composite literal of external " + t.Go, Span: span}
		}
		fieldType, err := b.typeOf(fieldGoType, span)
		if err != nil {
			return nil, err
		}
		value, err := b.buildExprAs(kv.Value, fieldType)
		if err != nil {
			return nil, err
		}
		fields = append(fields, key.Name)
		values = append(values, value)
		fieldTypes = append(fieldTypes, fieldType)
	}
	obligation := b.unit.AddExternalType(t.Pkg, t.Named)
	symbol := obligation.AddLiteralShape(fields, fieldTypes)
	b.use("externLit")
	return &ExternLit{T: t, Symbol: symbol, Values: values}, nil
}

// sameUnderlyingNamedPointers reports whether two pointer types point to
// named STRUCT types with IDENTICAL underlyings (type MutableNode Node):
// their generated classes are field-for-field identical, so the
// conversion is identity at the carrier.
func sameUnderlyingNamedPointers(fromGo, toGo types.Type) bool {
	fromPtr, fromOK := types.Unalias(fromGo).Underlying().(*types.Pointer)
	toPtr, toOK := types.Unalias(toGo).Underlying().(*types.Pointer)
	if !fromOK || !toOK {
		return false
	}
	fromNamed, fromOK := types.Unalias(fromPtr.Elem()).(*types.Named)
	toNamed, toOK := types.Unalias(toPtr.Elem()).(*types.Named)
	if !fromOK || !toOK {
		return false
	}
	if _, isStruct := fromNamed.Underlying().(*types.Struct); !isStruct {
		return false
	}
	return types.Identical(fromNamed.Underlying(), toNamed.Underlying())
}

// buildExternFieldRead lowers one exported-field read of an external
// struct value (or pointer to one) to its typed stub obligation.
func (b *builder) buildExternFieldRead(base Expr, field string, goType types.Type, span Span) (Expr, error) {
	extern := base.Type()
	if extern.Kind == KindPointer {
		extern = *extern.Elem
	}
	fieldType, err := b.typeOf(goType, span)
	if err != nil {
		return nil, err
	}
	obligation := b.unit.AddExternalType(extern.Pkg, extern.Named)
	symbol := obligation.AddFieldGet(field, fieldType)
	b.use("externFieldRead")
	return &ExternFieldRead{X: base, Symbol: symbol, Pkg: extern.Pkg, T: fieldType}, nil
}

// buildExternOwnedConversion bridges value conversions between an OWNED
// named struct and an EXTERNAL named struct with IDENTICAL underlyings
// (type CacheHashKey xxh3.Uint128): to the owned class via per-field
// read stubs, to the external via the keyed-literal constructor stub —
// both typed fail-closed obligations the emulation layer implements.
func (b *builder) buildExternOwnedConversion(x Expr, to Type, fromGo, toGo types.Type, span Span) (Expr, bool, error) {
	fromNamed, fromOK := types.Unalias(fromGo).(*types.Named)
	toNamed, toOK := types.Unalias(toGo).(*types.Named)
	if !fromOK || !toOK {
		return nil, false, nil
	}
	structType, isStruct := fromNamed.Underlying().(*types.Struct)
	if !isStruct || !types.Identical(fromNamed.Underlying(), toNamed.Underlying()) {
		return nil, false, nil
	}
	switch {
	case x.Type().Kind == KindExternal && to.Kind == KindStruct:
		obligation := b.unit.AddExternalType(x.Type().Pkg, x.Type().Named)
		out := &ExternToOwned{X: x, To: to}
		for i := range structType.NumFields() {
			field := structType.Field(i)
			if !field.Exported() {
				return nil, false, nil
			}
			fieldType, err := b.typeOf(field.Type(), span)
			if err != nil {
				return nil, true, err
			}
			out.FieldSymbols = append(out.FieldSymbols, obligation.AddFieldGet(field.Name(), fieldType))
		}
		b.use("externFieldRead")
		return out, true, nil
	case x.Type().Kind == KindStruct && to.Kind == KindExternal:
		obligation := b.unit.AddExternalType(to.Pkg, to.Named)
		fields := make([]string, 0, structType.NumFields())
		var fieldTypes []Type
		values := make([]Expr, 0, structType.NumFields())
		for i := range structType.NumFields() {
			field := structType.Field(i)
			if !field.Exported() {
				return nil, false, nil
			}
			fieldType, err := b.typeOf(field.Type(), span)
			if err != nil {
				return nil, true, err
			}
			fields = append(fields, field.Name())
			fieldTypes = append(fieldTypes, fieldType)
			values = append(values, &FieldLoad{X: x, Field: field.Name(), T: fieldType})
		}
		symbol := obligation.AddLiteralShape(fields, fieldTypes)
		b.use("externLit")
		return &ExternLit{T: to, Symbol: symbol, Values: values}, true, nil
	}
	return nil, false, nil
}
