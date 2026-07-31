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

func LabelIndex(value Label, index int) byte {
	return value[index]
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

func CountVariableShift(count Other) Count {
	return 1 << count
}

func CountVariableShiftUpdate(count Other) Count {
	var value Count
	value |= 1 << count
	return value
}

func FoldedCount() Count {
	return Count(2) + 3
}

func LocalTypes(value int32) int32 {
	type (
		Count int32
		Alias = Count
	)
	var left Count = Count(value)
	var right Alias = left
	return int32(left + right)
}

func CountUpdate(value Count) Count {
	value++
	value += 2
	value *= 3
	value--
	value &= 15
	value |= 16
	value ^= 3
	value &^= 1
	value <<= 1
	value >>= 1
	return value
}

func DefinedBuiltins(left, right Count, label Label) (Count, Count, int) {
	return min(left, right), max(left, right), len(label)
}

func CountPointer(value Count) Count {
	target := new(Count)
	*target = value
	(*target)++
	return *target
}

func CountSwitch(value Count) int32 {
	switch value {
	case 1:
		return 10
	case 2, 3:
		return 20
	default:
		return 30
	}
}

func CountArrayValues() ([2]Count, [2]Count, bool) {
	original := [2]Count{1, 2}
	copied := original
	copied[0]++
	return original, copied, original == [2]Count{1, 2}
}

func CountArrayCompoundOrder() Count {
	values := [1]Count{1}
	next := func() Count {
		values[0] = 10
		return 2
	}
	values[0] += next()
	return values[0]
}

func CountSliceValues() []Count {
	values := []Count{1, 2}
	values = append(values, 3)
	values[0]++
	copied := make([]Count, len(values))
	copy(copied, values)
	return copied
}

func CountMapValues() (Label, Label, bool) {
	values := map[Count]Label{
		1: "one",
		2: "two",
	}
	key := Count(2)
	found, ok := CountMapLookup(values, key)
	missing := values[Count(3)]
	return found, missing, ok
}

func CountMapLookup(
	values map[Count]Label,
	key Count,
) (Label, bool) {
	value, ok := values[key]
	return value, ok
}
