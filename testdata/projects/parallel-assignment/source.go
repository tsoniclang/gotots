package parallelassignment

func SwapLeft(left, right int32) int32 {
	left, right = right, left
	return left
}

func Rotate(current, next int32) int32 {
	current, previous := next, current
	return previous
}

func Declare(left, right int32) int32 {
	first, second := left, right
	return first + second
}

func Shadow(value int32) int32 {
	if true {
		value, previous := value+1, value
		return value + previous
	}
	return 0
}

func Accumulate(total, delta int32) int32 {
	total += delta
	return total
}
