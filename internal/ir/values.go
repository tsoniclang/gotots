// Constant and zero-value spelling for the reviewed carrier set.
package ir

import (
	"fmt"
	"go/constant"
	"math/big"
	"strconv"
)

func constValue(v constant.Value, t Type, span Span) (string, error) {
	switch t.Kind {
	case KindBool:
		return fmt.Sprintf("%v", constant.BoolVal(v)), nil
	case KindString:
		// The byte-string carrier holds one Go byte per JS code unit, so
		// the literal spells each byte: printable ASCII directly, all
		// other bytes (including non-UTF-8) as \xNN escapes.
		return byteStringLiteral(constant.StringVal(v)), nil
	case KindFloat32, KindFloat64:
		f, _ := constant.Float64Val(v)
		return strconv.FormatFloat(f, 'g', -1, 64), nil
	default:
		if !t.Kind.Integer() {
			return "", &Unsupported{Kind: KindConstantOfType, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "constant of type " + t.Go, Span: span}
		}
		text := v.ExactString()
		if _, ok := new(big.Int).SetString(text, 10); !ok {
			return "", &Unsupported{Kind: KindNonIntegralIntegerConstant, Code: "GOTOTS_UNSUPPORTED_EXPRESSION", Construct: "non-integral integer constant " + text, Span: span}
		}
		return text, nil
	}
}

// zeroValue materializes the Go zero value for a reviewed type.
func zeroValue(t Type, span Span) (Expr, error) {
	switch {
	case t.Kind == KindBool:
		return &Const{T: t, Value: "false"}, nil
	case t.Kind == KindString:
		return &Const{T: t, Value: `""`}, nil
	case t.Kind == KindIface && t.TypeParamName != "":
		// The zero of a type parameter comes from the instantiation's
		// zero factory, passed as a trailing parameter.
		return &ParamZero{T: t}, nil
	case t.Kind.Nilable():
		return &NilConst{T: t}, nil
	case t.Kind == KindStruct, t.Kind == KindArray:
		return &StructZero{T: t}, nil
	case t.Kind == KindUnit:
		return &Const{T: t, Value: "0"}, nil
	case t.Kind == KindExternal:
		return &ExternZero{T: t}, nil
	case t.Kind.Integer(), t.Kind.Float():
		return &Const{T: t, Value: "0"}, nil
	}
	return nil, &Unsupported{Kind: KindZeroValueOf, Code: "GOTOTS_UNSUPPORTED_TYPE", Construct: "zero value of " + t.Go, Span: span}
}
