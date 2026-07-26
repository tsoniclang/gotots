package integerconstants

func Small() int64 {
	return 42
}

func BeyondSafe() int64 {
	return 9007199254740993
}

func Maximum() int64 {
	return 9223372036854775807
}

func Minimum() int64 {
	return -9223372036854775808
}

func WideAdd(left, right int64) int64 {
	return left + right
}

func NativeAdd(left, right int) int {
	return left + right
}

func WideLess(left, right int64) bool {
	return left < right
}

func WideSwitch(value int64) int64 {
	switch value {
	case 0:
		return 1
	default:
		return 2
	}
}

func WideCompound(value, delta int64) int64 {
	value += delta
	return value
}

func WideIncrement(value int64) int64 {
	value++
	return value
}
