package multipleresults

func Pair(value int) (int, bool) {
	return value + 1, value >= 0
}

func Forward(value int) (int, bool) {
	return Pair(value)
}

func Consume(value int) int {
	next, positive := Pair(value)
	if positive {
		return next
	}
	return value
}

func Reassign(value int) int {
	next := value
	positive := false
	next, positive = Pair(value)
	if positive {
		return next
	}
	return value
}

func KeepFirst(value int) int {
	next, _ := Pair(value)
	return next
}

func Discard(value int) int {
	Pair(value)
	return value
}

func Numbers(value int) (int, int) {
	return value, value + 2
}

func Add(left, right int) int {
	return left + right
}

func AddPair(value int) int {
	return Add(Numbers(value))
}
