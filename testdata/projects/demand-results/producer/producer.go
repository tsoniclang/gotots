package producer

func Pair(value int32) (int32, bool) {
	return value + 1, value == 0
}

func Unused(value int32) int32 {
	return value + 100
}
