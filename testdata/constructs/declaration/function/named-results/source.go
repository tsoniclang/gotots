package namedresults

type Box struct {
	Value int32
}

func Next(value int32) (next int32, ok bool) {
	next = value + 1
	ok = value >= 0
	return
}

func Single(value int32) (result int32) {
	result = value * 2
	return
}

func Explicit(value int32) (result int32) {
	return value + 3
}

func Nested(value int32) (result int32) {
	transform := func(input int32) (result int32) {
		result = input + 4
		return
	}
	result = transform(value)
	return
}

func ZeroBox() (result Box) {
	return
}
