package ir

import (
	"go/types"
)

// typeOf resolves a go/types type into the reviewed IR type set. Types
// outside the subset produce GOTOTS_UNSUPPORTED_TYPE with the requesting
// construct's span. pkgPath scopes named struct types to the translated
// package: structs from other packages are external contracts, not local
// classes.
func (b *builder) typeOf(t types.Type, span Span) (Type, error) {
	if t == nil {
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "expression without type evidence", Span: span}
	}
	spelled := t.String()

	switch u := t.Underlying().(type) {
	case *types.Basic:
		kind, ok := basicKind(u)
		if !ok {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "basic type " + spelled, Span: span}
		}
		return Type{Kind: kind, Go: spelled}, nil

	case *types.Pointer:
		named, ok := types.Unalias(u.Elem()).(*types.Named)
		if !ok {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-named type " + spelled, Span: span}
		}
		if _, isStruct := named.Underlying().(*types.Struct); !isStruct {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to non-struct type " + spelled, Span: span}
		}
		if named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != b.pkgPath {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "pointer to type outside the translated package: " + spelled, Span: span}
		}
		element := Type{Kind: KindStruct, Go: named.String(), Named: named.Obj().Name()}
		return Type{Kind: KindPointer, Go: spelled, Named: named.Obj().Name(), Elem: &element}, nil

	case *types.Slice:
		element, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		if element.Kind == KindStruct {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "slice of struct values (element copy semantics)", Span: span}
		}
		return Type{Kind: KindSlice, Go: spelled, Elem: &element}, nil

	case *types.Map:
		key, err := b.typeOf(u.Key(), span)
		if err != nil {
			return Type{}, err
		}
		if !mapKeySupported(key.Kind) {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "map key type " + key.Go + " (Go key equality is not JS SameValueZero)", Span: span}
		}
		value, err := b.typeOf(u.Elem(), span)
		if err != nil {
			return Type{}, err
		}
		return Type{Kind: KindMap, Go: spelled, Key: &key, Elem: &value}, nil

	case *types.Struct:
		named, ok := types.Unalias(t).(*types.Named)
		if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != b.pkgPath {
			return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "struct type " + spelled, Span: span}
		}
		// Struct values are reviewed only behind pointers and receivers;
		// the caller decides whether a bare struct kind is admissible.
		return Type{Kind: KindStruct, Go: spelled, Named: named.Obj().Name()}, nil
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
// Map's SameValueZero over the canonical carriers: strings, booleans, and
// canonical-range integers/bigints. Floats are excluded (NaN semantics
// differ) and composite keys need generated hashing.
func mapKeySupported(kind Kind) bool {
	return kind == KindString || kind == KindBool || kind.Integer()
}
