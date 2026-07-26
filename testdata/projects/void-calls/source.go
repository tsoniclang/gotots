package voidcalls

func Touch(value int32) {
	if value > 0 {
		return
	}
}

func Identity(value int32) int32 {
	return value
}

func Run(value int32) int32 {
	Touch(value)
	Identity(value)
	return value
}
