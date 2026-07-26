package basicexpressions

func Arithmetic(value int64) int64 {
	return (value - 3) * 2
}

func WrapAdd(value int64) int64 {
	return value + 1
}

func WrapSubtract(value int64) int64 {
	return value - 1
}

func WrapMultiply(value int64) int64 {
	return value * 2
}

func IntWrapAdd(value int) int {
	return value + 1
}

func IntWrapSubtract(value int) int {
	return value - 1
}

func IntWrapMultiply(value int) int {
	return value * 2
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
