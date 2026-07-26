package parallelassignment

func SwapLeft(left, right int) int {
	left, right = right, left
	return left
}

func Rotate(current, next int) int {
	current, previous := next, current
	return previous
}

func Declare(left, right int) int {
	first, second := left, right
	return first + second
}

func Shadow(value int) int {
	if true {
		value, previous := value+1, value
		return value + previous
	}
	return 0
}
