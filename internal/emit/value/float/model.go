package float

import (
	"go/ast"
	"go/constant"
	"go/types"
	"math"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// Carrier describes a Go floating-point type's exact target representation. Both
// float32 and float64 ride the TypeScript `number` type; the width records the
// IEEE-754 rounding a float32 value must respect.
type Carrier struct {
	alias api.PrimitiveAlias
	bits  uint8
}

// Describe resolves a float32/float64 type to its carrier. Untyped and non-float
// types are not floats and return false.
func Describe(sourceType types.Type) (Carrier, bool) {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return Carrier{}, false
	}
	switch basic.Kind() {
	case types.Float32:
		return Carrier{alias: api.PrimitiveFloat32, bits: 32}, true
	case types.Float64:
		return Carrier{alias: api.PrimitiveFloat64, bits: 64}, true
	default:
		return Carrier{}, false
	}
}

func (c Carrier) Alias() api.PrimitiveAlias {
	return c.alias
}

func (c Carrier) Bits() uint8 {
	return c.bits
}

// FormatConstant materializes a constant's exact value at the carrier's
// representation as a decimal magnitude plus a sign. A float32 constant is first
// rounded to its nearest float32 (as Go does at compile time), then widened to
// the float64 that the target `number` literal must hold, so the emitted value
// is bit-identical to Go's float32 result — never the shorter float32 spelling,
// which would denote a different float64. The go/constant value carries full
// precision, so the source spelling is never read.
func FormatConstant(
	carrier Carrier,
	value constant.Value,
) (magnitude string, negative bool, ok bool) {
	if value == nil {
		return "", false, false
	}
	if value.Kind() != constant.Float && value.Kind() != constant.Int {
		return "", false, false
	}
	var wide float64
	switch carrier.bits {
	case 32:
		narrow, _ := constant.Float32Val(value)
		wide = float64(narrow)
	case 64:
		wide, _ = constant.Float64Val(value)
	default:
		return "", false, false
	}
	if math.IsInf(wide, 0) || math.IsNaN(wide) {
		return "", false, false
	}
	negative = math.Signbit(wide)
	magnitude = strconv.FormatFloat(math.Abs(wide), 'g', -1, 64)
	return magnitude, negative, true
}

// EmitConstant emits a floating-point constant as a target numeric literal at
// its carrier's exact representation.
func EmitConstant(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value constant.Value,
) (api.ExpressionEmission, error) {
	carrier, ok := Describe(targetType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	magnitude, negative, ok := FormatConstant(carrier, value)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target := tsgo.Expression(
		context.Factory().NumericLiteral(magnitude, tsgo.TokenFlagsNone),
	)
	if negative {
		target = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
			target,
		)
	}
	return api.DirectExpression(target), nil
}
