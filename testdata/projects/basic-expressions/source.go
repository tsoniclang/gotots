package basicexpressions

func Arithmetic(value int32) int32 {
	return (value - 3) * 2
}

func WrapAdd(value int32) int32 {
	return value + 1
}

func WrapSubtract(value int32) int32 {
	return value - 1
}

func WrapMultiply(value int32) int32 {
	return value * 2
}

func Increment(value int32) int32 {
	value++
	return value
}

func Decrement(value int32) int32 {
	value--
	return value
}

func Compare(left, right int32) (bool, bool, bool, bool, bool, bool) {
	return left == right,
		left != right,
		left < right,
		left <= right,
		left > right,
		left >= right
}

func Logic(left, right bool) bool {
	return (left && !right) || (!left && right)
}

func Never() bool {
	for {
	}
}

func ShortCircuitAnd() bool {
	return false && Never()
}

func ShortCircuitOr() bool {
	return true || Never()
}
