package wave3expressions

type Count int32
type Pair [2]int32
type Counts []int32
type Table map[int32]int32

type Box struct {
	Value int32
	Pair  Pair
}

type NestedZero struct {
	Value int32
}

type CompositeZero struct {
	Explicit int32
	Nested   [2]NestedZero
}

func compositeWithNestedZero() int32 {
	value := CompositeZero{Explicit: 31}
	var zero CompositeZero
	return value.Explicit + value.Nested[0].Value + zero.Nested[1].Value
}

type Flag bool
type Text string
type Float32 float32
type Float64 float64
type Complex64 complex64

func numberOperators(left, right int32) int32 {
	return left + right +
		(left - right) +
		left*right +
		(left & right) +
		(left | right) +
		(left ^ right) +
		(left &^ right) +
		(left << 3) +
		(left >> 3)
}

func definedOperators(left, right Count) Count {
	left += right
	left -= right
	left *= right
	left &= right
	left |= right
	left ^= right
	left &^= right
	left <<= 1
	left >>= 1
	left++
	left--
	return +left + -right + ^left
}

func stores(
	box Box,
	array Pair,
	slice Counts,
	table Table,
	pointer *int32,
) int32 {
	box.Value += 1
	array[0] -= 1
	slice[0] *= 2
	table[0] += 3
	*pointer ^= 4
	return box.Value + array[0] + slice[0] + table[0] + *pointer
}

func variadic(prefix int32, values ...int32) int32 {
	total := prefix
	for index := int32(0); index < int32(len(values)); index++ {
		total += values[index]
	}
	return total
}

func calls(values Counts) int32 {
	return variadic(1) + variadic(1, 2, 3) + variadic(1, values...)
}

func first[T any](value T, _ ...any) T {
	return value
}

func valueAndInterface() (int32, bool) {
	return 5, true
}

func variadicInterfaceTuple() int32 {
	return first(valueAndInterface())
}

func builtins(values Counts, table Table) int32 {
	values = append(values, 1, 2)
	values = append(values, Counts{3, 4}...)
	copied := make(Counts, len(values), cap(values)+1)
	count := copy(copied, values)
	delete(table, 0)
	clear(table)
	clear(copied)
	return int32(len(copied) + cap(copied) + count + len(table))
}

func shortCircuit(left bool, right func() bool) bool {
	return left && right() || !left && right()
}

func aggregate(values Counts) (Pair, *Pair, Box) {
	pair := Pair(values)
	pointer := (*Pair)(values)
	box := Box{Value: pair[0], Pair: pair}
	return pair, pointer, box
}

func expressionNew(value int32) *int32 {
	return new(value)
}

func staticNew() *Box {
	return new(Box)
}

func slices(value Counts) Counts {
	return value[:len(value):cap(value)]
}

func definedBasicOperators(
	flag Flag,
	text Text,
	left32 Float32,
	right32 Float32,
	left64 Float64,
	right64 Float64,
	leftComplex Complex64,
	rightComplex Complex64,
) (Flag, Text, Float32, Float64, Complex64, bool) {
	return !flag,
		text + "!",
		-left32 + right32,
		left64 / right64,
		leftComplex * rightComplex,
		leftComplex == rightComplex
}

func aggregateLiterals() ([3][2]int32, [][]int32) {
	array := [...][2]int32{{1, 2}, {3, 4}, {5, 6}}
	slices := [][]int32{{1, 2}, {3, 4}}
	return array, slices
}

func sliceEveryOperand(
	array [4]int32,
	pointer *[4]int32,
	slice Counts,
	text string,
) ([]int32, []int32, Counts, string) {
	return array[1:3], pointer[1:3:4], slice[1:3:4], text[1:3]
}

func stringSliceBuiltins(
	destination []byte,
	source string,
) (int32, []byte) {
	count := copy(destination, source)
	destination = append(destination, source...)
	return int32(count), destination
}

func containerMeasures(
	array [4]int32,
	pointer *[4]int32,
	slice Counts,
	table Table,
	text string,
) int32 {
	return int32(
		len(array) + cap(array) +
			len(pointer) + cap(pointer) +
			len(slice) + cap(slice) +
			len(table) + len(text),
	)
}

func constructContainers(
	length int32,
	capacity int32,
) (Counts, Table) {
	return make(Counts, length, capacity), make(Table, length)
}

func complexBuiltins(
	realPart float32,
	imaginaryPart float32,
) (complex64, float32, float32, float32, float32) {
	value := complex(realPart, imaginaryPart)
	return value,
		real(value),
		imag(value),
		min(realPart, imaginaryPart),
		max(realPart, imaginaryPart)
}

func moreStores(
	box *Box,
	array *Pair,
	slice Counts,
	table Table,
) int32 {
	box.Value |= 1
	array[0] &^= 2
	slice[0] <<= 1
	slice[1] >>= 1
	table[0]++
	table[1]--
	return box.Value +
		array[0] +
		slice[0] +
		slice[1] +
		table[0] +
		table[1]
}

func Audit() (
	int32,
	int32,
	int32,
	int32,
	int32,
	bool,
	int32,
	int32,
	int32,
	bool,
	string,
	float32,
	float64,
	float32,
	float32,
) {
	pointerValue := int32(3)
	storeResult := stores(
		Box{Value: 1},
		Pair{2, 3},
		Counts{4},
		Table{0: 5},
		&pointerValue,
	)
	rightCalls := int32(0)
	logical := shortCircuit(false, func() bool {
		rightCalls++
		return true
	})
	pair, pairPointer, box := aggregate(Counts{8, 9})
	newValue := expressionNew(11)
	newBox := staticNew()
	flag, text, float32Value, float64Value, complexValue, equal :=
		definedBasicOperators(
			false,
			"ok",
			1.5,
			2.5,
			9,
			4,
			complex(2, 3),
			complex(2, 3),
		)
	array, nestedSlices := aggregateLiterals()
	pointedArray := [4]int32{5, 6, 7, 8}
	arraySlice, pointerSlice, sliceSlice, textSlice := sliceEveryOperand(
		[4]int32{1, 2, 3, 4},
		&pointedArray,
		Counts{9, 10, 11, 12},
		"abcd",
	)
	copyCount, bytes := stringSliceBuiltins(
		[]byte{0, 0, 0},
		"xy",
	)
	measure := containerMeasures(
		[4]int32{},
		&pointedArray,
		Counts{1, 2},
		Table{1: 2},
		"abc",
	)
	createdSlice, createdTable := constructContainers(2, 4)
	complexResult, realPart, imaginaryPart, minimum, maximum :=
		complexBuiltins(3, -4)
	storeTail := moreStores(
		&Box{Value: 2},
		&Pair{7, 8},
		Counts{2, 8},
		Table{0: 4, 1: 6},
	)
	return numberOperators(9, 4),
		int32(definedOperators(7, 2)),
		storeResult + compositeWithNestedZero(),
		calls(Counts{2, 3}) + variadicInterfaceTuple(),
		builtins(Counts{1, 2}, Table{0: 9}),
		logical && rightCalls == 1,
		pair[0] + (*pairPointer)[1] + box.Value,
		*newValue + newBox.Value,
		int32(len(slices(Counts{1, 2, 3}))) +
			array[2][1] +
			int32(len(nestedSlices)) +
			arraySlice[0] +
			pointerSlice[0] +
			sliceSlice[0] +
			copyCount +
			int32(len(bytes)) +
			measure +
			int32(len(createdSlice)+len(createdTable)) +
			storeTail,
		bool(flag) || equal,
		string(text) + textSlice,
		float32(float32Value),
		float64(float64Value),
		real(complexValue) + realPart + real(complexResult),
		imag(complexValue) + imaginaryPart + minimum + maximum
}
