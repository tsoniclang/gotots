package definedbasic

type Count int32
type Other int32
type Label string
type Switch bool
type Ratio float64
type Narrow float32
type Signal complex128
type Alias = Count

const TypedConstant Count = 11
const UntypedConstant = 13

func CountFromInt(value int32) Count {
	return Count(value)
}

func IntFromCount(value Count) int32 {
	return int32(value)
}

func OtherFromCount(value Count) Other {
	return Other(value)
}

func CountFromOther(value Other) Count {
	return Count(value)
}

func AliasIdentity(value Alias) Count {
	return value
}

func CountZero() Count {
	var value Count
	return value
}

func CountArithmetic(left, right Count) (Count, Count, Count) {
	return left + right, left - right, left * right
}

func CountBits(left, right Count) (Count, Count, Count, Count) {
	return left & right, left | right, left ^ right, left &^ right
}

func CountUnary(value Count) (Count, Count, Count) {
	return +value, -value, ^value
}

func CountOrder(left, right Count) (bool, bool, bool, bool, bool, bool) {
	return left == right,
		left != right,
		left < right,
		left <= right,
		left > right,
		left >= right
}

func LabelFromString(value string) Label {
	return Label(value)
}

func StringFromLabel(value Label) string {
	return string(value)
}

func LabelJoin(left, right Label) Label {
	return left + right
}

func LabelOrder(left, right Label) (bool, bool, bool, bool, bool, bool) {
	return left == right,
		left != right,
		left < right,
		left <= right,
		left > right,
		left >= right
}

func SwitchFromBool(value bool) Switch {
	return Switch(value)
}

func BoolFromSwitch(value Switch) bool {
	return bool(value)
}

func SwitchNot(value Switch) Switch {
	return !value
}

func RatioFromFloat(value float64) Ratio {
	return Ratio(value)
}

func FloatFromRatio(value Ratio) float64 {
	return float64(value)
}

func RatioArithmetic(left, right Ratio) (Ratio, Ratio, Ratio, Ratio) {
	return left + right, left - right, left * right, left / right
}

func NarrowFromFloat(value float64) Narrow {
	return Narrow(value)
}

func FloatFromNarrow(value Narrow) float64 {
	return float64(value)
}

func NarrowAdd(left, right Narrow) Narrow {
	return left + right
}

func SignalFromComplex(value complex128) Signal {
	return Signal(value)
}

func SignalFromParts(realPart, imaginaryPart float64) Signal {
	return Signal(complex(realPart, imaginaryPart))
}

func ComplexFromSignal(value Signal) complex128 {
	return complex128(value)
}

func SignalProduct(left, right Signal) Signal {
	return left * right
}

func SignalEqual(left, right Signal) bool {
	return left == right
}

func ConstantValue() Count {
	return TypedConstant
}

func UntypedConstantValue() Count {
	return UntypedConstant
}

func CountWithLiteral(value Count) Count {
	return value + 2
}

func FoldedCount() Count {
	return Count(2) + 3
}
