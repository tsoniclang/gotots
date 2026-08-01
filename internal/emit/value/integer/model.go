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
		return sizedCarrier(sizes, types.Typ[types.Int], true)
	case types.Uint:
		return sizedCarrier(sizes, types.Typ[types.Uint], false)
	case types.Uintptr:
		return sizedCarrier(sizes, types.Typ[types.Uintptr], false)
	default:
		return Carrier{}, false
	}
}

func newCarrier(alias api.PrimitiveAlias, width uint8, signed bool) Carrier {
	return Carrier{alias: alias, width: width, signed: signed}
}

func sizedCarrier(
	sizes types.Sizes,
	sourceType types.Type,
	signed bool,
) (Carrier, bool) {
	switch sizes.Sizeof(sourceType) {
	case 4:
		if signed {
			return newCarrier(api.PrimitiveInt32, 32, true), true
		}
		return newCarrier(api.PrimitiveUint32, 32, false), true
	case 8:
		if signed {
			return newCarrier(api.PrimitiveInt64, 64, true), true
		}
		return newCarrier(api.PrimitiveUint64, 64, false), true
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
	return representation == api.IntegerRepresentationNumber &&
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
