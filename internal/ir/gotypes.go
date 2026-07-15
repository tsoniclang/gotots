package ir

import (
	"go/types"
)

// typeOf resolves a go/types type into the reviewed IR type set. Types
// outside the subset produce GOTOTS_UNSUPPORTED_TYPE with the requesting
// construct's span.
func typeOf(t types.Type, span Span) (Type, error) {
	if t == nil {
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "expression without type evidence", Span: span}
	}
	spelled := t.String()
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "non-basic type " + spelled, Span: span}
	}
	// Named basic types (type MyInt int32) carry their basic semantics;
	// their nominal identity is preserved in the Go spelling.
	var kind Kind
	switch basic.Kind() {
	case types.Bool, types.UntypedBool:
		kind = KindBool
	case types.String, types.UntypedString:
		kind = KindString
	case types.Int8:
		kind = KindInt8
	case types.Int16:
		kind = KindInt16
	case types.Int32:
		kind = KindInt32
	case types.Int64:
		kind = KindInt64
	case types.Int, types.UntypedInt:
		kind = KindInt
	case types.Uint8:
		kind = KindUint8
	case types.Uint16:
		kind = KindUint16
	case types.Uint32:
		kind = KindUint32
	case types.Uint64:
		kind = KindUint64
	case types.Uint:
		kind = KindUint
	case types.Uintptr:
		kind = KindUintptr
	case types.Float32:
		kind = KindFloat32
	case types.Float64, types.UntypedFloat:
		kind = KindFloat64
	default:
		return Type{}, &Unsupported{Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "basic type " + spelled, Span: span}
	}
	return Type{Kind: kind, Go: spelled}, nil
}
