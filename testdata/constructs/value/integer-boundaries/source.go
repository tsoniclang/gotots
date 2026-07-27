package integerboundaries

type Named int32

func NumberDivide(left, right int32) int32 {
	return left / right
}

func NumberRemainder(left, right int32) int32 {
	return left % right
}

func NumberInt64Bits(left, right int64) int64 {
	return left & right
}

func VariableShift(value int32, count uint8) int32 {
	return value << count
}

func Narrow(value int64) int8 {
	return int8(value)
}

func ChangeSign(value int8) uint64 {
	return uint64(value)
}

func NamedValue(value Named) Named {
	return value
}

func UnsafeNumber() int64 {
	return 9007199254740992
}

func UnsafeConversion() int64 {
	return int64(9007199254740992)
}
