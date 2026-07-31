package integerfamily

func Int8(left, right int8) int8 {
	var zero int8
	copy := left
	if copy == zero {
		return 1
	}
	return copy + right
}

func Int16(left, right int16) int16 {
	return left - right
}

func Int32(left, right int32) int32 {
	return left * right
}

func Int64(left, right int64) int64 {
	return left + right
}

func Uint8(left, right uint8) uint8 {
	return left + right
}

func Uint16(left, right uint16) uint16 {
	return left - right
}

func Uint32(left, right uint32) uint32 {
	return left * right
}

func Uint64(left, right uint64) uint64 {
	return left + right
}

func NativeInt(value int) int {
	return value + 1
}

func NativeUint(value uint) uint {
	return value + 1
}

func PointerUint(value uintptr) uintptr {
	return value + 1
}

func Byte(value byte) byte {
	return value + 1
}

func Rune(value rune) rune {
	return value + 1
}

func ConstantConversion() uint16 {
	return uint16(65535)
}

func NumberBits8(left, right int8) (int8, int8, int8, int8) {
	return left & right, left | right, left ^ right, left &^ right
}

func NumberBits16(left, right uint16) (uint16, uint16, uint16, uint16) {
	return left & right, left | right, left ^ right, left &^ right
}

func NumberBits32(left, right uint32) (uint32, uint32, uint32, uint32) {
	return left & right, left | right, left ^ right, left &^ right
}

func NumberShifts(value int32) (int32, int32) {
	return value << 3, value >> 2
}

func NumberUnsignedShift(value uint32) (uint32, uint32) {
	return value << 1, value >> 3
}

func NumberVariableShift(value int32, count uint8) (int32, int32) {
	return value << count, value >> count
}

func NumberVariableSignedShift(value int32, count int32) int32 {
	return value << count
}

func NumberVariableUnsignedShift(value uint32, count uint8) (uint32, uint32) {
	return value << count, value >> count
}

type DefinedShiftCount uint8

func DefinedShift(value int64, count DefinedShiftCount) (int64, int64) {
	return value << count, value >> count
}

func NumberUnary(value int32) (int32, int32, int32) {
	return +value, -value, ^value
}

func NumberUnaryUint(value uint32) uint32 {
	return ^value
}

func NumberWideBits(left, right int64) (int64, int64, int64, int64) {
	return left & right, left | right, left ^ right, left &^ right
}

func NumberWideShifts(value int64) (int64, int64) {
	return value << 3, value >> 2
}

func NumberWideUnary(value int) int {
	return ^value
}

func UntypedBooleanNot(left int32, right int32) bool {
	return !(left <= right && right <= 10)
}

func UnsignedComplement8(value uint8) uint8 {
	return ^value
}

func BigSigned(left, right int64) (int64, int64, int64, int64, int64, int64) {
	return left / right, left % right, left & right, left | right, left ^ right, left &^ right
}

func BigUnsigned(left, right uint64) (uint64, uint64, uint64, uint64) {
	return left / right, left % right, left & right, left ^ right
}

func BigShifts(value int64) (int64, int64) {
	return value << 2, value >> 3
}

func BigVariableShift(value int64, count uint8) (int64, int64) {
	return value << count, value >> count
}

func BigVariableSignedShift(value int64, count int32) int64 {
	return value << count
}

func BigVariableUnsignedShift(value uint64, count uint8) (uint64, uint64) {
	return value << count, value >> count
}

func BigUnary(value int64) (int64, int64, int64) {
	return +value, -value, ^value
}

func WidenSigned(value int8) int64 {
	return int64(value)
}

func WidenUnsigned(value uint32) int64 {
	return int64(value)
}

func CompareSigned(left, right int32) (bool, bool, bool, bool, bool, bool) {
	return left == right,
		left != right,
		left < right,
		left <= right,
		left > right,
		left >= right
}

func CompareUnsigned(left, right uint64) (bool, bool, bool, bool, bool, bool) {
	return left == right,
		left != right,
		left < right,
		left <= right,
		left > right,
		left >= right
}
