package sink

var Count int32

type marker struct{}

func (value marker) init() {}

func Mark(value int32) int32 {
	Count = Count*10 + value
	return Count
}

func Pair() (int32, int32) {
	return Mark(4), Mark(5)
}

func Read() int32 {
	return Count
}
