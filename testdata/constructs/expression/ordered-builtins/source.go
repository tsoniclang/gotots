package orderedbuiltins

func MaxInt32(first, second, third int32) int32 {
	return max(first, second, third)
}

func MinUint64(first, second, third uint64) uint64 {
	return min(first, second, third)
}

func MaxFloat32(first, second, third float32) float32 {
	return max(first, second, third)
}

func MinFloat64(first, second, third float64) float64 {
	return min(first, second, third)
}

func MaxString(first, second, third string) string {
	return max(first, second, third)
}

func MinString(first, second, third string) string {
	return min(first, second, third)
}

func One(value int32) int32 {
	return max(value)
}

func Mixed(value int32) int32 {
	return max(value, 7, -2)
}

func ConstantInteger() int32 {
	return max(1, 7, 3)
}

func ConstantFloat() float32 {
	return min(float32(1.5), float32(-2.25), float32(4))
}

func ConstantString() string {
	return max("a", "z", "m")
}

var EvaluationOrder string

func marked(label string, value int32) int32 {
	EvaluationOrder = EvaluationOrder + label
	return value
}

func OrderedMax() (int32, string) {
	EvaluationOrder = ""
	value := max(marked("a", 1), marked("b", 3), marked("c", 2))
	return value, EvaluationOrder
}

func pair() (int32, int32) {
	EvaluationOrder = EvaluationOrder + "p"
	return 4, 0
}

func first(value, ignored int32) int32 {
	return value
}

func OrderedWithPrerequisite() (int32, string) {
	EvaluationOrder = ""
	value := max(marked("a", 1), first(pair()), marked("c", 2))
	return value, EvaluationOrder
}
