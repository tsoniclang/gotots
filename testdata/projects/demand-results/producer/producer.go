package producer

func Pair(value int) (int, bool) {
	return value + 1, value == 0
}

func Unused(value int) int {
	return value + 100
}
