package boolflow

func Run(input bool) bool {
	current := false
	if !input {
		current = Flip(true)
	} else {
		current = Same(input, true)
	}
	return current
}

func Flip(value bool) bool {
	return !value
}

func Same(left, right bool) bool {
	return left == right
}
