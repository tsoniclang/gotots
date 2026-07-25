package boolmulti

func Run(input bool) bool {
	return flip(input)
}

func Again(input bool) bool {
	return flip(flip(input))
}

func identity(input bool) bool {
	return input
}
