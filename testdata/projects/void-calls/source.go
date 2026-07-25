package voidcalls

func Touch(value int) {
	if value > 0 {
		return
	}
}

func Identity(value int) int {
	return value
}

func Run(value int) int {
	Touch(value)
	Identity(value)
	return value
}
