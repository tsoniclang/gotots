// Slice semantic oracles: shared backing, capacity reuse on append,
// reslicing, nil versus empty, and Go's exact bounds panics — the
// conservative GoSlice carrier proven byte-identical against Go.
package translate_test

import "testing"

func TestOracleSliceBasics(t *testing.T) {
	runOracle(t, `package fixture

func LiteralLenCap() (int, int, int64) {
	values := []int64{10, 20, 30}
	return int(len(values)), int(cap(values)), values[1]
}

func MakeLenCapZero() (int, int, int32) {
	values := make([]int32, 2, 5)
	return int(len(values)), int(cap(values)), values[0]
}

func SetAndGet() int64 {
	values := make([]int64, 3)
	values[0] = 5
	values[1] = values[0] * 2
	values[2] = values[0] + values[1]
	return values[0]*100 + values[1]*10 + values[2]
}

func NilVersusEmpty() (bool, bool, int, int) {
	var nilSlice []int32
	empty := []int32{}
	return nilSlice == nil, empty == nil, int(len(nilSlice)), int(len(empty))
}

func RangeSum() int64 {
	values := []int64{1, 2, 3, 4, 5}
	total := int64(0)
	for _, value := range values {
		total = total + value
	}
	return total
}

func RangeIndexOnly() int {
	values := []string{"a", "b", "c"}
	count := 0
	for range values {
		count++
	}
	for index := range values {
		count = count + index
	}
	return count
}
`)
}

func TestOracleSliceAliasing(t *testing.T) {
	runOracle(t, `package fixture

func SharedBackingMutation() (int32, int32) {
	base := make([]int32, 3, 4)
	base[0] = 1
	base[1] = 2
	view := base[1:3]
	view[0] = 99
	return base[1], view[0]
}

func ResliceLenCap() (int, int, int, int) {
	base := make([]int32, 2, 6)
	view := base[1:2]
	return int(len(view)), int(cap(view)), int(len(base)), int(cap(base))
}

func AppendWithinCapacityAliases() (int32, int32, int) {
	base := make([]int32, 1, 3)
	base[0] = 4
	view := append(base, 5)
	view[0] = 8
	return base[0], view[1], int(cap(view))
}

func AppendBeyondCapacityAllocates() (int32, int32, int) {
	base := make([]int32, 1, 1)
	base[0] = 4
	grown := append(base, 5)
	grown[0] = 8
	return base[0], grown[0], int(len(grown))
}

func AppendToNil() (int, int64) {
	var values []int64
	values = append(values, 7)
	values = append(values, 8, 9)
	return int(len(values)), values[0] + values[1] + values[2]
}

func ChainedReslice() int64 {
	base := []int64{1, 2, 3, 4, 5}
	middle := base[1:4]
	inner := middle[1:2]
	inner[0] = 100
	return base[2]
}
`)
}

func TestOracleSlicePanics(t *testing.T) {
	runOracle(t, `package fixture

func IndexOutOfRange() int64 {
	values := []int64{1, 2, 3}
	index := 5
	return values[index]
}

func IndexOnNil() int32 {
	var values []int32
	index := 0
	return values[index]
}

func ResliceBeyondCapacity() int {
	values := make([]int32, 1, 3)
	high := 5
	view := values[0:high]
	return int(len(view))
}

func StoreOutOfRange() int32 {
	values := make([]int32, 2)
	index := 2
	values[index] = 1
	return values[0]
}
`)
}
