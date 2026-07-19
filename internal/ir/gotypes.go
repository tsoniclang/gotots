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
		if core := constraintCore(param); core != nil {
			switch core.(type) {
			case *types.Slice, *types.Map:
				// A CORE-TYPED parameter (S ~[]E, M ~map[K]V): every
				// binding shares the core carrier exactly — the parameter
				// ERASES to it (no factory participation; slice/map
				// derivations are kind-driven and total).
				resolved, err := b.typeOf(core, span)
				if err != nil {
					return Type{}, err
				}
				resolved.Go = spelled
				resolved.ErasedParamName = param.Obj().Name()
				return resolved, nil
			}
		}
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
			if named.Obj().Pkg() != nil {
				// A pointer to a named carrier type (owned or external —
				// a non-struct external named erases to its value
				// carrier): a named ARRAY keeps identity (the array IS
				// the handle); every boxable carrier takes a mutable
				// cell, the name preserved for method dispatch.
				element, err := b.typeOf(named, span)
				if err != nil {
					return Type{}, err
				}
				if element.Kind != KindArray && !boxable(element.Kind) {
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
		if named.Obj().Parent() != named.Obj().Pkg().Scope() {
			if mentionsTypeParamType(u) {
				// A local struct capturing an enclosing generic's type
				// parameters has per-instantiation shape: fail closed.
				return Type{}, &Unsupported{Kind: KindStructType, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "local struct type capturing type parameters " + spelled, Span: span}
			}
			// A LOCAL named struct: no package-level declaration exists —
			// its class synthesizes through the anonymous-struct pipeline
			// (the NAMED canonical identity keeps distinct locals
			// distinct; Go forbids methods on local types).
			return b.localNamedStructType(named, u, spelled, span)
		}
		// Struct values are reviewed only behind pointers and receivers;
		// the caller decides whether a bare struct kind is admissible.
		out := Type{Kind: KindStruct, Go: spelled, Named: named.Obj().Name(), Pkg: named.Obj().Pkg().Path(),
			// Uncomparable tracks whether the CLASS carries goEq$ (the
			// generated exact equality) — the emitted surface, which is
			// stricter than Go's spec comparability. An eq factory over a
			// binding without goEq$ fails closed loudly, never silently.
			Uncomparable: !b.structEqComparable(named),
			KeyEncodable: b.structKeyEncodable(named, span)}
		if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			for i := range named.TypeParams().Len() {
				out.ClassKeyParams = append(out.ClassKeyParams, b.unit.ParamRequiresKeyOp(named.Origin().Obj(), i))
				out.HardKeyedParams = append(out.HardKeyedParams, b.unit.ParamRequiresSVZKey(named.Origin().Obj(), i))
			}
			if named.TypeArgs() != nil && named.TypeArgs().Len() > 0 {
				out.MapFamilyEnc = b.instanceFamilyEnc(named)
			}
		}
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
					!mentionsTypeParamType(goArg) && !mapKeySupported(arg.Kind) &&
					!(arg.Kind == KindStruct && arg.KeyEncodable) &&
					!(arg.Kind == KindIface && b.ifaceKeyMembersEncodable(goArg, span)) {
					return Type{}, &Unsupported{Kind: KindGenericInstantiationOutsideAdmittedKeyFamily, Code: "GOTOTS_UNSUPPORTED_TYPE",
						Construct: "generic instantiation outside the admitted key family (" + spelled + ": " + goArg.String() + " is not an admitted map key)", Span: span}
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
			for i := range named.TypeParams().Len() {
				out.ClassKeyParams = append(out.ClassKeyParams, b.unit.ParamRequiresKeyOp(named.Obj(), i))
				out.HardKeyedParams = append(out.HardKeyedParams, b.unit.ParamRequiresSVZKey(named.Obj(), i))
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


// constraintCore computes a type parameter's core type: the single
// underlying every term of its constraint shares, or nil.
func constraintCore(param *types.TypeParam) types.Type {
	iface, ok := param.Constraint().Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	var core types.Type
	for i := range iface.NumEmbeddeds() {
		union, isUnion := iface.EmbeddedType(i).(*types.Union)
		if !isUnion {
			continue
		}
		for t := range union.Len() {
			u := union.Term(t).Type().Underlying()
			if core == nil {
				core = u
			} else if !types.Identical(core, u) {
				return nil
			}
		}
	}
	return core
}


// coreErasedParam reports whether a type parameter erases to its core
// carrier (slice/map cores drop out of the emitted generic surface).
func coreErasedParam(param *types.TypeParam) bool {
	core := constraintCore(param)
	if core == nil {
		return false
	}
	switch core.(type) {
	case *types.Slice, *types.Map:
		return true
	}
	return false
}
