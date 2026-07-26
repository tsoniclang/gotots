package multipleresults

func Pair(value int32) (int32, bool) {
	return value + 1, value >= 0
}

func Forward(value int32) (int32, bool) {
	return Pair(value)
}

func Consume(value int32) int32 {
	next, positive := Pair(value)
	if positive {
		return next
	}
	return value
}

func Reassign(value int32) int32 {
	next := value
	positive := false
	next, positive = Pair(value)
	if positive {
		return next
	}
	return value
}

func KeepFirst(value int32) int32 {
	next, _ := Pair(value)
	return next
}

func Discard(value int32) int32 {
	Pair(value)
	return value
}

func Numbers(value int32) (int32, int32) {
	return value, value + 2
}

func Add(left, right int32) int32 {
	return left + right
}

func AddPair(value int32) int32 {
	return Add(Numbers(value))
}
