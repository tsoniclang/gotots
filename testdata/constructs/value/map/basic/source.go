package mapvalues

var Global map[int32]int32
var Seeded = map[int32]int32{5: 6}

func Missing() int32 {
	values := make(map[int32]int32)
	return values[7]
}

func Lookup(key int32) (int32, bool) {
	values := map[int32]int32{1: 0, 2: 20}
	value, ok := values[key]
	return value, ok
}

func Alias() int32 {
	values := make(map[int32]int32)
	alias := values
	alias[3] = 31
	return values[3]
}

func ThroughCall() int32 {
	values := make(map[int32]int32, 4)
	Identity(values)[4] = 41
	return values[4]
}

func Identity(values map[int32]int32) map[int32]int32 {
	return values
}

func DeleteAndLen() (int32, int) {
	values := map[int32]int32{1: 10, 2: 20}
	delete(values, 1)
	delete(values, 7)
	var missing map[int32]int32
	delete(missing, 1)
	return values[1], len(values)
}

func BoolKey() int32 {
	values := map[bool]int32{false: 4, true: 9}
	return values[false] + values[true]
}

func LiteralOrder() int32 {
	var next int32
	step := func() int32 {
		next++
		return next
	}
	values := map[int32]int32{step(): step(), step(): step()}
	return values[1]*1000 + values[3]*10 + next
}

func NilLength() int {
	var values map[int32]int32
	return len(values)
}

func ExplicitNil() map[int32]int32 {
	return nil
}

func ResetToNil() int {
	values := make(map[int32]int32)
	values[1] = 2
	values = nil
	return len(values)
}

func NilComparisons() (bool, bool) {
	var values map[int32]int32
	wasNil := values == nil
	values = make(map[int32]int32)
	return wasNil, values != nil
}

func MakeSized(size int) map[int32]int32 {
	return make(map[int32]int32, size)
}

func SizeEvaluated() int {
	var calls int
	size := func() int {
		calls++
		return -1
	}
	values := make(map[int32]int32, size())
	return calls + len(values)
}

func PackageState(value int32) (int32, int32) {
	if Global == nil {
		Global = make(map[int32]int32)
	}
	Global[1] = value
	return Global[1], Seeded[5]
}

func NilWrite() {
	var values map[int32]int32
	values[1] = 2
}
