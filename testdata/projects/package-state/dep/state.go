package dep

type Cell struct {
	Value int32
}

var Trace int32
var __proto__ int32
var Dormant int32
var Empty Cell
var Filled Cell = Cell{Value: 4}

func mark(value int32) int32 {
	Trace = Trace*10 + value
	__proto__++
	return value
}

func Snapshot() int32 {
	return A*10000 +
		B*1000 +
		Trace*10 +
		hidden +
		__proto__ +
		Empty.Value +
		Filled.Value
}
