package census

import (
	"fmt"
	"go/types"
)

// typeClass names the semantic operand class of a type for range/index
// census purposes, resolving named types to their underlying form. The
// classification is exhaustive: a type variant outside the reviewed set is
// an error, never an "unknown" or "other" bucket — it means the pinned
// toolchain's type universe has a shape this census has not been reviewed
// against.
func typeClass(t types.Type) (string, error) {
	if t == nil {
		return "", fmt.Errorf("operand has no recorded type")
	}
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return "typeParam", nil
	}
	switch u := t.Underlying().(type) {
	case *types.Slice:
		return "slice", nil
	case *types.Array:
		return "array", nil
	case *types.Map:
		return "map", nil
	case *types.Chan:
		return "chan", nil
	case *types.Signature:
		return "func", nil
	case *types.Interface:
		return "interface", nil
	case *types.Pointer:
		if _, ok := u.Elem().Underlying().(*types.Array); ok {
			return "pointerToArray", nil
		}
		return "pointer", nil
	case *types.Struct:
		return "struct", nil
	case *types.Tuple:
		return "tuple", nil
	case *types.Basic:
		switch {
		case u.Info()&types.IsString != 0:
			return "string", nil
		case u.Info()&types.IsInteger != 0:
			return "integer", nil
		case u.Info()&types.IsFloat != 0:
			return "float", nil
		case u.Info()&types.IsComplex != 0:
			return "complex", nil
		case u.Info()&types.IsBoolean != 0:
			return "bool", nil
		case u.Kind() == types.UnsafePointer:
			return "unsafePointer", nil
		case u.Kind() == types.Invalid:
			return "", fmt.Errorf("operand has invalid type")
		default:
			return "basic:" + u.Name(), nil
		}
	default:
		return "", fmt.Errorf("unreviewed type variant %T (%s)", u, t)
	}
}

// namedKind classifies a declared type's underlying shape for the
// declaration contract, exhaustively.
func namedKind(t types.Type) (string, error) {
	switch u := t.Underlying().(type) {
	case *types.Struct:
		return "struct", nil
	case *types.Interface:
		return "interface", nil
	case *types.Slice:
		return "slice", nil
	case *types.Array:
		return "array", nil
	case *types.Map:
		return "map", nil
	case *types.Chan:
		return "chan", nil
	case *types.Signature:
		return "func", nil
	case *types.Pointer:
		return "pointer", nil
	case *types.Basic:
		return "basic:" + u.Name(), nil
	default:
		return "", fmt.Errorf("unreviewed underlying type variant %T (%s)", u, t)
	}
}
