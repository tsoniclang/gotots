package ir

import (
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/typeid"

	"golang.org/x/tools/go/packages"
)

// ResolveType resolves one go/types type through the reviewed type set:
// a standalone entry to the builder's resolver for declaration-level
// checks.
func ResolveType(p *packages.Package, sourceDir string, unit Scope, t types.Type, pos token.Pos) (Type, error) {
	return ResolveTypeIn(p, sourceDir, unit, t, pos, nil)
}

// ResolveTypeIn resolves a type within a type-parameter binder
// environment (e.g. a generic external function's own parameters), so a
// component type referencing those parameters canonicalizes rather than
// failing closed.
func ResolveTypeIn(p *packages.Package, sourceDir string, unit Scope, t types.Type, pos token.Pos, binders *typeid.Binders) (Type, error) {
	b := &builder{
		fset:       p.Fset,
		info:       p.TypesInfo,
		pkgPath:    p.PkgPath,
		sourceDir:  sourceDir,
		unit:       unit,
		operations: map[string]bool{},
		sites:      &[]UnsupportedSite{},
		binders:    binders,
	}
	return b.typeOf(t, b.span(pos))
}

// typeOf resolves a go/types type into the reviewed IR type set. Types
// outside the subset produce GOTOTS_UNSUPPORTED_TYPE with the requesting
// construct's span. Named struct types are scoped to the translated unit:
// structs from packages outside it are external contracts, not generated
// classes.
// typeOf resolves one Go type and stamps its canonical semantic
// identity (typeid.Canonical) — the ONE key every evidence ledger uses.
func (b *builder) typeOf(t types.Type, span Span) (Type, error) {
	resolved, err := b.typeOfInner(t, span)
	if err != nil {
		return resolved, err
	}
	if resolved.Canon == "" {
		// Canonical is a total contract: an unhandled type form fails
		// closed. Component types referencing the declaration's type
		// parameters canonicalize within the builder's binder environment.
		canon, err := b.canonical(t)
		if err != nil {
			return Type{}, &Unsupported{Kind: KindNestedError, Code: "GOTOTS_TYPE_UNSUPPORTED",
				Construct: err.Error(), Span: span}
		}
		resolved.Canon = canon
	}
	return resolved, nil
}

// canonical renders a type's identity within the builder's binder
// environment (the declaration's type parameters), or the empty
// environment when the builder is not inside a generic declaration.
func (b *builder) canonical(t types.Type) (string, error) {
	if b.binders != nil {
		return b.binders.Canonical(t)
	}
	return typeid.Canonical(t)
}

func (b *builder) typeOfInner(t types.Type, span Span) (Type, error) {
	if t == nil {
		return Type{}, &Unsupported{Kind: KindExpressionWithoutTypeEvidence, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "expression without type evidence", Span: span}
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
			return Type{}, &Unsupported{Kind: KindBasicType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "basic type " + spelled, Span: span}
		}
		return Type{Kind: kind, Go: spelled}, nil

	case *types.Pointer:
		if named, ok := types.Unalias(u.Elem()).(*types.Named); ok {
			if _, isStruct := named.Underlying().(*types.Struct); isStruct {
				if named.Obj().Pkg() == nil {
					return Type{}, &Unsupported{Kind: KindPointerToTypeOutsideTheTranslatedUnit, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to type outside the translated unit: " + spelled, Span: span}
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
					return Type{}, &Unsupported{Kind: KindPointerToNonStructType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-struct type " + spelled, Span: span}
				}
				return Type{Kind: KindPointer, Go: spelled, Named: named.Obj().Name(), Pkg: named.Obj().Pkg().Path(), Elem: &element}, nil
			}
			return Type{}, &Unsupported{Kind: KindPointerToNonStructType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-struct type " + spelled, Span: span}
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
			return Type{}, &Unsupported{Kind: KindPointerToNonNamedType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-named type " + spelled, Span: span}
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
			// A FLOAT key carries Go's exact float-key semantics through the
			// dedicated GoFMap carrier (NaN inserts fresh unretrievable
			// entries; +0/-0 coincide).
			admitted := key.Kind.Float()
			admitted = admitted || key.Kind == KindStruct && b.structKeyEncodable(u.Key(), span)
			if !admitted && key.Kind == KindIface {
				// An INTERFACE key admits when every retained member's
				// dynamic key is pointer identity: the union's generated
				// $key function encodes discriminant + goKeyId, exactly
				// Go's (dynamic type, pointer) equality. Scalar/struct/
				// handle members stay fail-closed pending their encoders.
				admitted = b.ifaceKeyMembersEncodable(u.Key(), span)
			}
			if !admitted {
				// A type-parameter key hides behind its constraint
				// interface; the instantiation evidence decides it.
				if _, isParam := types.Unalias(u.Key()).(*types.TypeParam); isParam {
					admitted = b.typeParamKeySupported(u.Key(), span)
				}
			}
			if !admitted {
				return Type{}, &Unsupported{Kind: KindMapKeyType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "map key type " + key.Go + " (Go key equality is not JS SameValueZero)", Span: span}
			}
		}
		value, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		return Type{Kind: KindMap, Go: spelled, Key: &key, Elem: &value}, nil

	case *types.Signature:
		if u.TypeParams() != nil || u.Recv() != nil {
			return Type{}, &Unsupported{Kind: KindGenericFunctionType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "generic function type " + spelled, Span: span}
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
			return Type{}, &Unsupported{Kind: KindStructType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct type " + spelled, Span: span}
		}
		// Struct values are reviewed only behind pointers and receivers;
		// the caller decides whether a bare struct kind is admissible.
		out := Type{Kind: KindStruct, Go: spelled, Named: named.Obj().Name(), Pkg: named.Obj().Pkg().Path(),
			Uncomparable: !types.Comparable(t)}
		if named.TypeArgs() != nil {
			for i := range named.TypeArgs().Len() {
				goArg := named.TypeArgs().At(i)
				arg, err := b.typeOf(goArg, span)
				if err != nil {
					return Type{}, err
				}
				// The per-site key-family guard: a concrete binding of a
				// parameter the declaration keys a map by must be a
				// SameValueZero carrier — the declaration's GoMap carrier is
				// exact for every instantiation that exists in emitted code.
				// A free (parameter-mentioning) argument is guarded at its
				// own concrete sites.
				if b.unit.ParamRequiresSVZKey(named.Origin().Obj(), i) &&
					!mentionsTypeParamType(goArg) && !mapKeySupported(arg.Kind) {
					return Type{}, &Unsupported{Kind: KindGenericInstantiationOutsideAdmittedKeyFamily, Code: "GOTOTS_UNSUPPORTED_TYPE",
						Construct: "generic instantiation outside the admitted key family (" + spelled + ": " + goArg.String() + " is not a SameValueZero key)", Span: span}
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
		members, err := b.ifaceMembers(u, span)
		if err != nil {
			return Type{}, err
		}
		out.IfaceMembers = members
		ifaceID, err := b.canonicalIfaceID(u)
		if err != nil {
			return Type{}, err
		}
		out.IfaceID = ifaceID
		out.IfaceEmpty = u.NumMethods() == 0
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
		// The TYPE position is representable (an opaque nilable handle no
		// emitted code can construct); every channel OPERATION remains
		// typed-unimplemented at its own site.
		return Type{Kind: KindChan, Go: spelled}, nil

	}
	return Type{}, &Unsupported{Kind: KindType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "type " + spelled, Span: span}
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
	// Floats encode through the exact float-key rule: -0 folds onto +0
	// and every NaN encode is a fresh token, so a NaN-field key is
	// unretrievable — exactly Go.
	return t.Kind.Integer() || t.Kind.Float()
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
	// The declaration admits with the direct GoMap carrier: the prepass
	// recorded the SVZ-key REQUIREMENT for this parameter, and every
	// concrete instantiation site is guarded against it (a non-SVZ binding
	// fails closed AT ITS SITE), so the carrier stays exact for every
	// instantiation that exists in emitted code. The evidence is consulted
	// only to confirm the requirement machinery covers this declaration.
	_ = instances
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
// hold it. A nested comparable struct field composes through the field
// type's own goKey$, so the recursion descends into struct and
// array-of-struct fields.
func (b *builder) structKeyEncodable(keyType types.Type, span Span) bool {
	structType, ok := types.Unalias(keyType).Underlying().(*types.Struct)
	if !ok {
		return false
	}
	guard := keyType.String()
	if b.keyEncodableInProgress[guard] {
		// A cycle through member/field types: the definition is
		// COINDUCTIVE (assuming encodable is self-consistent — value
		// recursion terminates because pointer members encode by
		// identity), so the in-progress type counts as encodable.
		return true
	}
	if b.keyEncodableInProgress == nil {
		b.keyEncodableInProgress = map[string]bool{}
	}
	b.keyEncodableInProgress[guard] = true
	defer delete(b.keyEncodableInProgress, guard)
	for i := range structType.NumFields() {
		if !b.keyEncodableFieldType(structType.Field(i).Type(), span) {
			return false
		}
	}
	return true
}

// keyEncodableFieldType reports whether one struct field's Go type
// participates in the generated canonical key encoding. Scalars, byte
// strings, pointers (identity), and unit encode directly; a nested named
// struct encodes through its own goKey$ (recursion); a fixed array
// encodes element-wise. Floats (NaN, signed zeros), interfaces, external
// types, and maps/slices stay out — their Go equality is not injective
// onto a deterministic string.
func (b *builder) keyEncodableFieldType(goType types.Type, span Span) bool {
	fieldIR, err := b.typeOf(goType, span)
	if err != nil {
		return false
	}
	switch fieldIR.Kind {
	case KindStruct:
		return b.structKeyEncodable(goType, span)
	case KindArray:
		if arr, ok := types.Unalias(goType).Underlying().(*types.Array); ok {
			return b.keyEncodableFieldType(arr.Elem(), span)
		}
		return false
	case KindIface:
		// An interface FIELD encodes through the union's generated $key
		// (Go's dynamic-key equality) when every member is encodable.
		if fieldIR.TypeParamName != "" {
			return false
		}
		return b.ifaceKeyMembersEncodable(goType, span)
	}
	return KeyEncodableField(fieldIR)
}

// ifaceKeyMembersEncodable reports whether every member of an interface
// type used as a MAP KEY is exactly encodable: pointer identity, a
// scalar carrier (basic-underlying external types included), or a
// key-encodable struct value. An external HANDLE value member (opaque,
// compared by value in Go) has no exact encoding and fails closed. The
// EMPTY interface additionally admits: predeclared members are scalars
// and boxed composites either encode (identity/array-prim) or carry Go's
// exact unhashable panic.
func (b *builder) ifaceKeyMembersEncodable(goType types.Type, span Span) bool {
	iface, ok := goType.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	guard := "iface:" + goType.String()
	if b.keyEncodableInProgress[guard] {
		return true // coinductive: see structKeyEncodable
	}
	if b.keyEncodableInProgress == nil {
		b.keyEncodableInProgress = map[string]bool{}
	}
	b.keyEncodableInProgress[guard] = true
	defer delete(b.keyEncodableInProgress, guard)
	members, err := b.ifaceMembers(iface, span)
	if err != nil {
		return false
	}
	if iface.NumMethods() == 0 {
		// The empty interface: named members checked below; predeclared
		// and composite members are always encodable-or-panicking.
	} else if len(members) == 0 {
		return false
	}
	for _, member := range members {
		switch {
		case member.Pointer:
		case member.Extern && member.ExternCarrier != "":
		case member.Extern:
			return false // opaque handle value: no exact encoding
		case member.Struct:
			if !member.KeyEncodable {
				// An UNCOMPARABLE dynamic key panics in Go ("hash of
				// unhashable type") — the panic IS the exact encoding; a
				// comparable member without an encoding fails closed.
				if member.Eq == nil || member.Eq.Kind != EqUncomparable {
					return false
				}
			}
		default:
			// A named value carrier over a scalar: encodable by carrier.
		}
	}
	return true
}
