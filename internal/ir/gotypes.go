package ir

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// ResolveType resolves one go/types type through the reviewed type set:
// a standalone entry to the builder's resolver for declaration-level
// checks.
func ResolveType(p *packages.Package, sourceDir string, unit Scope, t types.Type, pos token.Pos) (Type, error) {
	b := &builder{
		fset:       p.Fset,
		info:       p.TypesInfo,
		pkgPath:    p.PkgPath,
		sourceDir:  sourceDir,
		unit:       unit,
		operations: map[string]bool{},
		sites:      &[]UnsupportedSite{},
	}
	return b.typeOf(t, b.span(pos))
}

// typeOf resolves a go/types type into the reviewed IR type set. Types
// outside the subset produce GOTOTS_UNSUPPORTED_TYPE with the requesting
// construct's span. Named struct types are scoped to the translated unit:
// structs from packages outside it are external contracts, not generated
// classes.
func (b *builder) typeOf(t types.Type, span Span) (Type, error) {
	if t == nil {
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "expression without type evidence", Span: span}
	}
	spelled := t.String()

	if param, isParam := types.Unalias(t).(*types.TypeParam); isParam {
		// A type parameter's underlying is its constraint interface; the
		// carrier is the opaque interface kind, spelled by the parameter
		// name so generic signatures stay generic.
		return Type{Kind: KindIface, Go: spelled, TypeParamName: param.Obj().Name()}, nil
	}

	switch u := t.Underlying().(type) {
	case *types.Basic:
		kind, ok := basicKind(u)
		if !ok {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "basic type " + spelled, Span: span}
		}
		return Type{Kind: kind, Go: spelled}, nil

	case *types.Pointer:
		if named, ok := types.Unalias(u.Elem()).(*types.Named); ok {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct {
				if named.Obj().Pkg() == nil {
					return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to type outside the translated unit: " + spelled, Span: span}
				}
				declaringPkg := named.Obj().Pkg().Path()
				if !b.unit.Owns(declaringPkg) {
					// A pointer to an external struct: the handle itself,
					// with undefined for nil (external handles have object
					// identity).
					element := Type{Kind: KindExternal, Go: named.String(), Named: named.Obj().Name(), Pkg: declaringPkg}
					return Type{Kind: KindPointer, Go: spelled, Named: named.Obj().Name(), Pkg: declaringPkg, Elem: &element}, nil
				}
				element, err := b.typeOf(named, span)
				if err != nil {
					return Type{}, err
				}
				return Type{Kind: KindPointer, Go: spelled, Named: named.Obj().Name(), Pkg: declaringPkg, Elem: &element}, nil
			}
			if named.Obj().Pkg() != nil && b.unit.Owns(named.Obj().Pkg().Path()) {
				// A pointer to an owned named carrier type: a mutable
				// cell, with the name preserved for method dispatch.
				element, err := b.typeOf(named, span)
				if err != nil {
					return Type{}, err
				}
				if !boxable(element.Kind) {
					return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-struct type " + spelled, Span: span}
				}
				return Type{Kind: KindPointer, Go: spelled, Named: named.Obj().Name(), Pkg: named.Obj().Pkg().Path(), Elem: &element}, nil
			}
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-struct type " + spelled, Span: span}
		}
		// A pointer to a non-named type: fixed arrays keep their carrier
		// identity (the handle is the pointer); every other reviewed
		// pointee is carried as a mutable cell created where the address
		// is taken.
		element, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		if element.Kind == KindStruct || element.Kind == KindExternal || element.Kind == KindTypeParam {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-named type " + spelled, Span: span}
		}
		return Type{Kind: KindPointer, Go: spelled, Elem: &element}, nil

	case *types.Slice:
		element, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		return Type{Kind: KindSlice, Go: spelled, Elem: &element}, nil

	case *types.Map:
		key, err := b.typeOf(u.Key(), span)
		if err != nil {
			return Type{}, err
		}
		if !mapKeySupported(key.Kind) {
			admitted := key.Kind == KindStruct && b.structKeyEncodable(u.Key(), span)
			if !admitted {
				// A type-parameter key hides behind its constraint
				// interface; the instantiation evidence decides it.
				if _, isParam := types.Unalias(u.Key()).(*types.TypeParam); isParam {
					admitted = b.typeParamKeySupported(u.Key(), span)
				}
			}
			if !admitted {
				return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "map key type " + key.Go + " (Go key equality is not JS SameValueZero)", Span: span}
			}
		}
		value, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		return Type{Kind: KindMap, Go: spelled, Key: &key, Elem: &value}, nil

	case *types.Signature:
		if u.TypeParams() != nil || u.Recv() != nil {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "generic function type " + spelled, Span: span}
		}
		// A variadic signature is exact through its carrier: the final
		// parameter is the packed slice, and every call site packs (or
		// spreads) through the call-building evidence.
		sig := &FuncSig{}
		for i := range u.Params().Len() {
			parameter, err := b.typeOf(u.Params().At(i).Type(), span)
			if err != nil {
				return Type{}, err
			}
			sig.Params = append(sig.Params, parameter)
		}
		for i := range u.Results().Len() {
			result, err := b.typeOf(u.Results().At(i).Type(), span)
			if err != nil {
				return Type{}, err
			}
			sig.Results = append(sig.Results, result)
		}
		return Type{Kind: KindFunc, Go: spelled, Sig: sig}, nil

	case *types.Struct:
		named, ok := types.Unalias(t).(*types.Named)
		if !ok || named.Obj().Pkg() == nil || !b.unit.Owns(named.Obj().Pkg().Path()) {
			if !ok && u.NumFields() == 0 {
				// The anonymous empty struct is the unit type; named empty
				// structs keep their identity (methods, pointers, rtti).
				return Type{Kind: KindUnit, Go: spelled}, nil
			}
			if !ok {
				return b.anonStructType(u, spelled, span)
			}
			if ok && named.Obj().Pkg() != nil {
				// A named struct outside the unit is an opaque external
				// handle under the external-contract policy; the stub
				// module carries its value-semantics contract.
				b.unit.AddExternalType(named.Obj().Pkg().Path(), named.Obj().Name())
				return Type{Kind: KindExternal, Go: spelled, Named: named.Obj().Name(), Pkg: named.Obj().Pkg().Path()}, nil
			}
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct type " + spelled, Span: span}
		}
		// Struct values are reviewed only behind pointers and receivers;
		// the caller decides whether a bare struct kind is admissible.
		out := Type{Kind: KindStruct, Go: spelled, Named: named.Obj().Name(), Pkg: named.Obj().Pkg().Path()}
		if named.TypeArgs() != nil {
			for i := range named.TypeArgs().Len() {
				arg, err := b.typeOf(named.TypeArgs().At(i), span)
				if err != nil {
					return Type{}, err
				}
				out.TypeArgs = append(out.TypeArgs, arg)
			}
		} else if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			// The generic type referenced inside its own declaration
			// scope (a method receiver, a recursive field): its arguments
			// are its own type parameters.
			for i := range named.TypeParams().Len() {
				arg, err := b.typeOf(named.TypeParams().At(i), span)
				if err != nil {
					return Type{}, err
				}
				out.TypeArgs = append(out.TypeArgs, arg)
			}
		}
		return out, nil
	case *types.Interface:
		out := Type{Kind: KindIface, Go: spelled}
		if named, isNamed := types.Unalias(t).(*types.Named); isNamed && named.Obj().Pkg() != nil {
			out.Named = named.Obj().Name()
			out.Pkg = named.Obj().Pkg().Path()
		}
		return out, nil

	case *types.Array:
		element, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		out := Type{Kind: KindArray, Go: spelled, Elem: &element, ArrayLen: u.Len()}
		if named, isNamed := types.Unalias(t).(*types.Named); isNamed && named.Obj().Pkg() != nil {
			out.Named = named.Obj().Name()
			out.Pkg = named.Obj().Pkg().Path()
		}
		return out, nil

	case *types.Chan:
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "channel type " + spelled, Span: span}

	}
	return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "type " + spelled, Span: span}
}

func basicKind(basic *types.Basic) (Kind, bool) {
	switch basic.Kind() {
	case types.Bool, types.UntypedBool:
		return KindBool, true
	case types.String, types.UntypedString:
		return KindString, true
	case types.Int8:
		return KindInt8, true
	case types.Int16:
		return KindInt16, true
	case types.Int32:
		return KindInt32, true
	case types.Int64:
		return KindInt64, true
	case types.Int, types.UntypedInt:
		return KindInt, true
	case types.Uint8:
		return KindUint8, true
	case types.Uint16:
		return KindUint16, true
	case types.Uint32:
		return KindUint32, true
	case types.Uint64:
		return KindUint64, true
	case types.Uint:
		return KindUint, true
	case types.Uintptr:
		return KindUintptr, true
	case types.Float32:
		return KindFloat32, true
	case types.Float64, types.UntypedFloat:
		return KindFloat64, true
	}
	return KindInvalid, false
}

// mapKeySupported admits key kinds whose Go equality coincides with JS
// Map's SameValueZero over the canonical carriers: strings, booleans,
// canonical-range integers/bigints, and pointers (Go pointer equality is
// exactly JS object identity, with undefined for nil). Floats are
// excluded (NaN semantics differ) and composite keys need generated
// hashing.
func mapKeySupported(kind Kind) bool {
	return kind == KindString || kind == KindBool || kind == KindPointer || kind.Integer() || kind == KindUnit
}

// KeyEncodableField reports whether one field type participates in the
// generated canonical key encoding (goKey$): kinds whose Go equality
// maps injectively onto a deterministic string component. Floats (NaN,
// signed zeros), interfaces, and nested named structs stay out.
func KeyEncodableField(t Type) bool {
	switch t.Kind {
	case KindString, KindBool, KindPointer, KindUnit:
		return true
	case KindArray:
		return KeyEncodableField(*t.Elem)
	}
	return t.Kind.Integer()
}

// typeParamKeySupported admits a type-parameter map key when every
// recorded instantiation of the enclosing generic function binds it to
// a carrier whose Go equality is JS SameValueZero — the closed-world
// evidence makes the direct Map carrier exact for all of them.
func (b *builder) typeParamKeySupported(keyType types.Type, span Span) bool {
	param, ok := types.Unalias(keyType).(*types.TypeParam)
	if !ok {
		return false
	}
	var instances [][]types.Type
	switch {
	case b.genericObj != nil:
		signature := b.genericObj.Type().(*types.Signature)
		if signature.TypeParams() == nil || param.Index() >= signature.TypeParams().Len() ||
			signature.TypeParams().At(param.Index()) != param {
			return false
		}
		instances = b.unit.GenericInstances(b.genericObj)
	case b.genericTypeObj != nil:
		typeParams := b.genericTypeObj.TypeParams()
		if typeParams == nil || param.Index() >= typeParams.Len() ||
			typeParams.At(param.Index()).Obj().Name() != param.Obj().Name() {
			return false
		}
		instances = b.unit.GenericTypeInstances(b.genericTypeObj.Obj())
	default:
		return false
	}
	if len(instances) == 0 {
		return false
	}
	for _, instance := range instances {
		if param.Index() >= len(instance) {
			return false
		}
		bound, err := b.typeOf(instance[param.Index()], span)
		if err != nil || !mapKeySupported(bound.Kind) {
			return false
		}
	}
	return true
}

// EqComparableField reports whether one field type participates in the
// generated exact equality (goEq$): direct === carriers (floats keep
// NaN and signed-zero semantics under ===), nested comparable structs,
// arrays of those, and interfaces (whose equality may panic, exactly
// Go). External fields and uncomparable kinds stay out.
func (b *builder) EqComparableField(t Type, goType types.Type) bool {
	switch t.Kind {
	case KindIface:
		// A type-parameter field's equality is instantiation-dependent
		// (=== for one instantiation, interface equality for another):
		// the shared goEq$ body cannot be exact, so the struct stays out.
		return t.TypeParamName == ""
	case KindString, KindBool, KindPointer, KindUnit, KindFloat32, KindFloat64:
		return true
	case KindArray:
		elemGo := goType
		if arr, ok := types.Unalias(goType).Underlying().(*types.Array); ok {
			elemGo = arr.Elem()
		}
		return b.EqComparableField(*t.Elem, elemGo)
	case KindStruct:
		return b.structEqComparable(goType)
	}
	return t.Kind.Integer()
}

// structEqComparable reports whether a struct's generated class carries
// goEq$ — exact field-wise equality over every field.
func (b *builder) structEqComparable(goType types.Type) bool {
	structType, ok := types.Unalias(goType).Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := range structType.NumFields() {
		field := structType.Field(i)
		fieldIR, err := b.typeOf(field.Type(), Span{})
		if err != nil || !b.EqComparableField(fieldIR, field.Type()) {
			return false
		}
	}
	return true
}

// structKeyEncodable reports whether a named struct key's fields are all
// encodable, so its class carries goKey$ and the keyed-map carrier can
// hold it.
func (b *builder) structKeyEncodable(keyType types.Type, span Span) bool {
	structType, ok := types.Unalias(keyType).Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := range structType.NumFields() {
		fieldType, err := b.typeOf(structType.Field(i).Type(), span)
		if err != nil || !KeyEncodableField(fieldType) {
			return false
		}
	}
	return true
}
