package representationcontract

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type NativeIntegerWidth uint8

const (
	NativeIntegerWidthInvalid NativeIntegerWidth = 0
	NativeIntegerWidth32      NativeIntegerWidth = 32
	NativeIntegerWidth64      NativeIntegerWidth = 64
)

func (w NativeIntegerWidth) Valid() bool {
	return w == NativeIntegerWidth32 || w == NativeIntegerWidth64
}

func NativeIntegerWidthFromSizes(sizes types.Sizes) (NativeIntegerWidth, error) {
	if sizes == nil {
		return NativeIntegerWidthInvalid, &NativeIntegerWidthError{}
	}
	switch sizes.Sizeof(types.Typ[types.Int]) {
	case 4:
		return NativeIntegerWidth32, nil
	case 8:
		return NativeIntegerWidth64, nil
	default:
		return NativeIntegerWidthInvalid, &NativeIntegerWidthError{}
	}
}

type IntegerRepresentation uint8

const (
	IntegerRepresentationInvalid IntegerRepresentation = iota
	IntegerRepresentationNumber
	IntegerRepresentationBigInt
	IntegerRepresentationFixed64BigInt
)

func (r IntegerRepresentation) Valid() bool {
	return r == IntegerRepresentationNumber ||
		r == IntegerRepresentationBigInt ||
		r == IntegerRepresentationFixed64BigInt
}

type ScalarABI struct {
	integer     IntegerRepresentation
	nativeWidth NativeIntegerWidth
}

func NewScalarABI(
	integer IntegerRepresentation,
	nativeWidth NativeIntegerWidth,
) (ScalarABI, error) {
	if !integer.Valid() {
		return ScalarABI{}, &IntegerRepresentationError{
			Representation: integer,
		}
	}
	if !nativeWidth.Valid() {
		return ScalarABI{}, &NativeIntegerWidthError{Width: nativeWidth}
	}
	return ScalarABI{integer: integer, nativeWidth: nativeWidth}, nil
}

func NewScalarABIFromSizes(
	integer IntegerRepresentation,
	sizes types.Sizes,
) (ScalarABI, error) {
	width, err := NativeIntegerWidthFromSizes(sizes)
	if err != nil {
		return ScalarABI{}, err
	}
	return NewScalarABI(integer, width)
}

func (a ScalarABI) Valid() bool {
	return a.integer.Valid() && a.nativeWidth.Valid()
}

func (a ScalarABI) IntegerRepresentation() IntegerRepresentation {
	return a.integer
}

func (a ScalarABI) NativeIntegerWidth() NativeIntegerWidth {
	return a.nativeWidth
}

func (a ScalarABI) Carrier(alias PrimitiveAlias) (IntegerCarrier, error) {
	return IntegerCarrierRepresentation(alias, a)
}

func (a ScalarABI) UsesBigInt(source types.Type) bool {
	alias, ok := PrimitiveAliasFor(source)
	if !ok {
		return false
	}
	carrier, err := a.Carrier(alias)
	return err == nil && carrier == IntegerCarrierBigInt
}

func (r IntegerRepresentation) String() string {
	switch r {
	case IntegerRepresentationNumber:
		return "number"
	case IntegerRepresentationBigInt:
		return "bigint"
	case IntegerRepresentationFixed64BigInt:
		return "fixed64-bigint"
	default:
		return fmt.Sprintf("integer-representation(%d)", r)
	}
}

func ParseIntegerRepresentation(value string) (IntegerRepresentation, error) {
	switch value {
	case "number":
		return IntegerRepresentationNumber, nil
	case "bigint":
		return IntegerRepresentationBigInt, nil
	case "fixed64-bigint":
		return IntegerRepresentationFixed64BigInt, nil
	default:
		return IntegerRepresentationInvalid, &IntegerRepresentationError{}
	}
}

type EvaluationOrder uint8

const (
	EvaluationOrderInvalid EvaluationOrder = iota
	EvaluationOrderDirect
	EvaluationOrderPreserveGo
)

func (o EvaluationOrder) Valid() bool {
	return o == EvaluationOrderDirect ||
		o == EvaluationOrderPreserveGo
}

func (o EvaluationOrder) String() string {
	switch o {
	case EvaluationOrderDirect:
		return "direct"
	case EvaluationOrderPreserveGo:
		return "preserve-go"
	default:
		return fmt.Sprintf("evaluation-order(%d)", o)
	}
}

type MethodReceiverABI uint8

const (
	MethodReceiverABIInvalid MethodReceiverABI = iota
	MethodReceiverABISourceRepresentation
	MethodReceiverABIContractDirect
)

func (a MethodReceiverABI) Valid() bool {
	return a == MethodReceiverABISourceRepresentation ||
		a == MethodReceiverABIContractDirect
}

func IntegerLiteral(
	factory tsgo.Factory,
	abi ScalarABI,
	alias PrimitiveAlias,
	decimal string,
) (tsgo.Expression, error) {
	if decimal == "" {
		return nil, &IntegerRepresentationError{
			Representation: abi.IntegerRepresentation(),
		}
	}
	carrier, err := IntegerCarrierRepresentation(alias, abi)
	if err != nil {
		return nil, err
	}
	switch carrier {
	case IntegerCarrierNumber:
		return factory.NumericLiteral(decimal, tsgo.TokenFlagsNone), nil
	case IntegerCarrierBigInt:
		return factory.BigIntLiteral(decimal+"n", tsgo.TokenFlagsNone), nil
	default:
		return nil, &IntegerRepresentationError{
			Representation: abi.IntegerRepresentation(),
		}
	}
}

type PrimitiveAlias uint8

const (
	PrimitiveInvalid PrimitiveAlias = iota
	PrimitiveBool
	PrimitiveInt8
	PrimitiveInt16
	PrimitiveInt32
	PrimitiveInt64
	PrimitiveUint8
	PrimitiveUint16
	PrimitiveUint32
	PrimitiveUint64
	PrimitiveString
	PrimitiveFloat32
	PrimitiveFloat64
	PrimitiveInt
	PrimitiveUint
	PrimitiveUintptr
)

func PrimitiveAliasFor(source types.Type) (PrimitiveAlias, bool) {
	basic, ok := types.Unalias(source).(*types.Basic)
	if !ok {
		return PrimitiveInvalid, false
	}
	switch basic.Kind() {
	case types.Bool:
		return PrimitiveBool, true
	case types.Int8:
		return PrimitiveInt8, true
	case types.Int16:
		return PrimitiveInt16, true
	case types.Int32:
		return PrimitiveInt32, true
	case types.Int64:
		return PrimitiveInt64, true
	case types.Uint8:
		return PrimitiveUint8, true
	case types.Uint16:
		return PrimitiveUint16, true
	case types.Uint32:
		return PrimitiveUint32, true
	case types.Uint64:
		return PrimitiveUint64, true
	case types.String:
		return PrimitiveString, true
	case types.Float32:
		return PrimitiveFloat32, true
	case types.Float64:
		return PrimitiveFloat64, true
	case types.Int:
		return PrimitiveInt, true
	case types.Uint:
		return PrimitiveUint, true
	case types.Uintptr:
		return PrimitiveUintptr, true
	default:
		return PrimitiveInvalid, false
	}
}

func PrimitiveTypeScriptType(
	source types.Type,
	abi ScalarABI,
) (string, bool) {
	alias, ok := PrimitiveAliasFor(source)
	if !ok {
		return "", false
	}
	_, keyword, err := PrimitiveAliasRepresentation(alias, abi)
	if err != nil {
		return "", false
	}
	switch keyword {
	case tsgo.KeywordTypeSyntaxKindBooleanKeyword:
		return "boolean", true
	case tsgo.KeywordTypeSyntaxKindStringKeyword:
		return "string", true
	case tsgo.KeywordTypeSyntaxKindNumberKeyword:
		return "number", true
	case tsgo.KeywordTypeSyntaxKindBigIntKeyword:
		return "bigint", true
	default:
		return "", false
	}
}

type IntegerCarrier uint8

const (
	IntegerCarrierInvalid IntegerCarrier = iota
	IntegerCarrierNumber
	IntegerCarrierBigInt
)

func (c IntegerCarrier) Valid() bool {
	return c == IntegerCarrierNumber || c == IntegerCarrierBigInt
}

func IntegerCarrierRepresentation(
	alias PrimitiveAlias,
	abi ScalarABI,
) (IntegerCarrier, error) {
	if !abi.IntegerRepresentation().Valid() {
		return IntegerCarrierInvalid, &IntegerRepresentationError{
			Representation: abi.IntegerRepresentation(),
		}
	}
	if !abi.NativeIntegerWidth().Valid() {
		return IntegerCarrierInvalid, &NativeIntegerWidthError{
			Width: abi.NativeIntegerWidth(),
		}
	}
	switch alias {
	case PrimitiveInt8,
		PrimitiveInt16,
		PrimitiveInt32,
		PrimitiveUint8,
		PrimitiveUint16,
		PrimitiveUint32:
		return IntegerCarrierNumber, nil
	case PrimitiveInt64, PrimitiveUint64:
		if abi.IntegerRepresentation() == IntegerRepresentationBigInt ||
			abi.IntegerRepresentation() == IntegerRepresentationFixed64BigInt {
			return IntegerCarrierBigInt, nil
		}
		return IntegerCarrierNumber, nil
	case PrimitiveInt, PrimitiveUint, PrimitiveUintptr:
		if abi.IntegerRepresentation() == IntegerRepresentationBigInt &&
			abi.NativeIntegerWidth() == NativeIntegerWidth64 {
			return IntegerCarrierBigInt, nil
		}
		return IntegerCarrierNumber, nil
	default:
		return IntegerCarrierInvalid, &PrimitiveAliasError{Alias: alias}
	}
}

func PrimitiveAliasRepresentation(
	alias PrimitiveAlias,
	abi ScalarABI,
) (string, tsgo.KeywordTypeSyntaxKind, error) {
	name, err := PrimitiveAliasName(alias)
	if err != nil {
		return "", 0, err
	}
	switch alias {
	case PrimitiveBool:
		return name, tsgo.KeywordTypeSyntaxKindBooleanKeyword, nil
	case PrimitiveString:
		return name, tsgo.KeywordTypeSyntaxKindStringKeyword, nil
	case PrimitiveFloat32, PrimitiveFloat64:
		return name, tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	case PrimitiveInt8,
		PrimitiveInt16,
		PrimitiveInt32,
		PrimitiveInt64,
		PrimitiveUint8,
		PrimitiveUint16,
		PrimitiveUint32,
		PrimitiveUint64,
		PrimitiveInt,
		PrimitiveUint,
		PrimitiveUintptr:
		carrier, err := IntegerCarrierRepresentation(alias, abi)
		if err != nil {
			return "", 0, err
		}
		keyword, err := integerKeyword(carrier)
		return name, keyword, err
	default:
		return "", 0, &PrimitiveAliasError{Alias: alias}
	}
}

func PrimitiveAliasName(alias PrimitiveAlias) (string, error) {
	switch alias {
	case PrimitiveBool:
		return "bool", nil
	case PrimitiveInt8:
		return "int8", nil
	case PrimitiveInt16:
		return "int16", nil
	case PrimitiveInt32:
		return "int32", nil
	case PrimitiveInt64:
		return "int64", nil
	case PrimitiveUint8:
		return "uint8", nil
	case PrimitiveUint16:
		return "uint16", nil
	case PrimitiveUint32:
		return "uint32", nil
	case PrimitiveUint64:
		return "uint64", nil
	case PrimitiveString:
		return "gostring", nil
	case PrimitiveFloat32:
		return "float32", nil
	case PrimitiveFloat64:
		return "float64", nil
	case PrimitiveInt:
		return "int", nil
	case PrimitiveUint:
		return "uint", nil
	case PrimitiveUintptr:
		return "uintptr", nil
	default:
		return "", &PrimitiveAliasError{Alias: alias}
	}
}

func integerKeyword(
	carrier IntegerCarrier,
) (tsgo.KeywordTypeSyntaxKind, error) {
	switch carrier {
	case IntegerCarrierNumber:
		return tsgo.KeywordTypeSyntaxKindNumberKeyword, nil
	case IntegerCarrierBigInt:
		return tsgo.KeywordTypeSyntaxKindBigIntKeyword, nil
	default:
		return 0, &IntegerCarrierError{Carrier: carrier}
	}
}

type IntegerCarrierError struct {
	Carrier IntegerCarrier
}

func (e *IntegerCarrierError) Error() string {
	return fmt.Sprintf("integer carrier %d is invalid", e.Carrier)
}

type PrimitiveAliasError struct {
	Alias PrimitiveAlias
}

func (e *PrimitiveAliasError) Error() string {
	return fmt.Sprintf("primitive alias %d is invalid", e.Alias)
}

type IntegerRepresentationError struct {
	Representation IntegerRepresentation
}

type NativeIntegerWidthError struct {
	Width NativeIntegerWidth
}

func (e *NativeIntegerWidthError) Error() string {
	return fmt.Sprintf("native integer width %d is invalid", e.Width)
}

func (e *IntegerRepresentationError) Error() string {
	return fmt.Sprintf(
		"integer representation %d is invalid",
		e.Representation,
	)
}
