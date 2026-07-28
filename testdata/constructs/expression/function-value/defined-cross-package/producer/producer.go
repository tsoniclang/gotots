package producer

type Transform func(int32) int32

func Increment(value int32) int32 {
	return value + 1
}

func Make() Transform {
	return Increment
}

func Apply(transform Transform, value int32) int32 {
	return transform(value)
}
