package registry

var Trace int32

func Mark(value int32) int32 {
	Trace = Trace*10 + value
	return Trace
}

func Read() int32 {
	return Trace
}
