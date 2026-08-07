package integer

import (
	"go/constant"
	"go/token"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type Carrier struct {
	alias  api.PrimitiveAlias
	width  uint8
	signed bool
}

func Describe(sizes types.Sizes, sourceType types.Type) (Carrier, bool) {
	if sizes == nil || sourceType == nil {
		return Carrier{}, false
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok {
		return Carrier{}, false
	}
	switch basic.Kind() {
	case types.Int8:
		return newCarrier(api.PrimitiveInt8, 8, true), true
	case types.Int16:
		return newCarrier(api.PrimitiveInt16, 16, true), true
	case types.Int32:
		return newCarrier(api.PrimitiveInt32, 32, true), true
	case types.Int64:
		return newCarrier(api.PrimitiveInt64, 64, true), true
	case types.Uint8:
		return newCarrier(api.PrimitiveUint8, 8, false), true
	case types.Uint16:
		return newCarrier(api.PrimitiveUint16, 16, false), true
	case types.Uint32:
		return newCarrier(api.PrimitiveUint32, 32, false), true
	case types.Uint64:
		return newCarrier(api.PrimitiveUint64, 64, false), true
	case types.Int:
		return nativeCarrier(sizes, api.PrimitiveInt, true)
	case types.Uint:
		return nativeCarrier(sizes, api.PrimitiveUint, false)
	case types.Uintptr:
		return nativeCarrier(sizes, api.PrimitiveUintptr, false)
	default:
		return Carrier{}, false
	}
}

func DescribeUnderlying(
	sizes types.Sizes,
	sourceType types.Type,
) (Carrier, bool) {
	if carrier, ok := Describe(sizes, sourceType); ok {
		return carrier, true
	}
	if sourceType == nil {
		return Carrier{}, false
	}
	basic, ok := types.Unalias(sourceType).Underlying().(*types.Basic)
	if !ok {
		return Carrier{}, false
	}
	return Describe(sizes, basic)
}

func newCarrier(alias api.PrimitiveAlias, width uint8, signed bool) Carrier {
	return Carrier{alias: alias, width: width, signed: signed}
}

func nativeCarrier(
	sizes types.Sizes,
	alias api.PrimitiveAlias,
	signed bool,
) (Carrier, bool) {
	switch sizes.Sizeof(types.Typ[types.Int]) {
	case 4:
		return newCarrier(alias, 32, signed), true
	case 8:
		return newCarrier(alias, 64, signed), true
	default:
		return Carrier{}, false
	}
}

func (c Carrier) Alias() api.PrimitiveAlias {
	return c.alias
}

func (c Carrier) Width() uint8 {
	return c.width
}

func (c Carrier) Signed() bool {
	return c.signed
}

func CarrierRepresentation(
	profile api.IntegerRepresentation,
	carrier Carrier,
) (api.IntegerCarrier, bool) {
	width := api.NativeIntegerWidth64
	if carrier.Alias() == api.PrimitiveInt ||
		carrier.Alias() == api.PrimitiveUint ||
		carrier.Alias() == api.PrimitiveUintptr {
		width = api.NativeIntegerWidth(carrier.Width())
	}
	abi, err := api.NewScalarABI(profile, width)
	if err != nil {
		return api.IntegerCarrierInvalid, false
	}
	representation, err := api.IntegerCarrierRepresentation(
		carrier.Alias(),
		abi,
	)
	return representation, err == nil
}

func UsesBigInt(
	profile api.IntegerRepresentation,
	carrier Carrier,
) bool {
	representation, ok := CarrierRepresentation(profile, carrier)
	return ok && representation == api.IntegerCarrierBigInt
}

func FormatConstant(
	representation api.IntegerRepresentation,
	carrier Carrier,
	value constant.Value,
) (magnitude string, negative bool, ok bool) {
	if !representation.Valid() ||
		value == nil ||
		value.Kind() != constant.Int ||
		carrier.width == 0 {
		return "", false, false
	}
	if carrier.signed {
		signed, exact := constant.Int64Val(value)
		if !exact || !fitsSigned(signed, carrier.width) {
			return "", false, false
		}
		if signed < 0 {
			return strconv.FormatUint(absolute(signed), 10), true, true
		}
		return strconv.FormatInt(signed, 10), false, true
	}
	unsigned, exact := constant.Uint64Val(value)
	if !exact || !fitsUnsigned(unsigned, carrier.width) {
		return "", false, false
	}
	return strconv.FormatUint(unsigned, 10), false, true
}

func fitsSigned(value int64, width uint8) bool {
	if width == 64 {
		return true
	}
	limit := int64(1) << (width - 1)
	return value >= -limit && value < limit
}

func fitsUnsigned(value uint64, width uint8) bool {
	return width == 64 || value < uint64(1)<<width
}

func absolute(value int64) uint64 {
	if value >= 0 {
		return uint64(value)
	}
	return uint64(-(value + 1)) + 1
}

func SupportsArithmetic(
	representation api.IntegerRepresentation,
	operator token.Token,
) bool {
	if !representation.Valid() {
		return false
	}
	switch operator {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
		return true
	default:
		return false
	}
}

func SupportsBitwise(
	representation api.IntegerRepresentation,
	carrier Carrier,
	operator token.Token,
) bool {
	switch operator {
	case token.AND, token.OR, token.XOR, token.AND_NOT:
	default:
		return false
	}
	return representation.Valid()
}

func SupportsShift(
	representation api.IntegerRepresentation,
	carrier Carrier,
	operator token.Token,
	count constant.Value,
) bool {
	if operator != token.SHL && operator != token.SHR {
		return false
	}
	if !representation.Valid() {
		return false
	}
	if count == nil || count.Kind() != constant.Int {
		return false
	}
	unsigned, exact := constant.Uint64Val(count)
	return exact && unsigned < uint64(carrier.width)
}

func SupportsVariableShift(
	representation api.IntegerRepresentation,
	carrier Carrier,
	operator token.Token,
) bool {
	if operator != token.SHL && operator != token.SHR {
		return false
	}
	return representation.Valid()
}

func SupportsUnary(
	representation api.IntegerRepresentation,
	carrier Carrier,
	operator token.Token,
) bool {
	switch operator {
	case token.ADD, token.SUB:
		return representation.Valid()
	case token.XOR:
		return representation.Valid()
	default:
		return false
	}
}

func UnsignedMask(carrier Carrier) (string, bool) {
	if carrier.signed || carrier.width == 0 {
		return "", false
	}
	if carrier.width == 64 {
		return strconv.FormatUint(^uint64(0), 10), true
	}
	return strconv.FormatUint(uint64(1)<<carrier.width-1, 10), true
}

func RequiresUint32Normalization(
	representation api.IntegerRepresentation,
	carrier Carrier,
) bool {
	target, ok := CarrierRepresentation(representation, carrier)
	return ok && target == api.IntegerCarrierNumber &&
		!carrier.signed &&
		carrier.width == 32
}

func CanConvertDirectly(source Carrier, target Carrier) bool {
	switch {
	case source.width == 0 || target.width == 0:
		return false
	case source.signed && target.signed:
		return target.width >= source.width
	case !source.signed && !target.signed:
		return target.width >= source.width
	case !source.signed && target.signed:
		return target.width > source.width
	default:
		return false
	}
}
