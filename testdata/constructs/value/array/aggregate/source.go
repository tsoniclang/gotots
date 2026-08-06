package aggregatearray

type Box struct {
	Value int32
}

type Boxes [2]Box
type Matrix [2][2]Box

type Phantom[T any] struct {
	_     [0]T
	Value int32
}

func genericZeroLengthPhantom[T any](value int32) int32 {
	phantom := Phantom[T]{Value: value}
	return phantom.Value
}

func GenericZeroLengthPhantom() int32 {
	return genericZeroLengthPhantom[Box](17)
}

func ZeroLengthPointerKey() int32 {
	key := [0]*Box{}
	values := map[[0]*Box]int32{key: 29}
	return values[key]
}

func NewBoxes(left, right int32) Boxes {
	return Boxes{{Value: left}, {Value: right}}
}

func ZeroFresh() (int32, int32) {
	var values [2]Box
	values[0].Value = 7
	return values[0].Value, values[1].Value
}

func CopyIsDeep() (int32, int32) {
	original := [2]Box{{Value: 1}, {Value: 2}}
	copied := original
	copied[0].Value = 9
	return original[0].Value, copied[0].Value
}

func NamedCopyIsDeep() (int32, int32) {
	original := Boxes{{Value: 3}, {Value: 4}}
	copied := original
	copied[1].Value = 8
	return original[1].Value, copied[1].Value
}

func NestedCopyIsDeep() (int32, int32) {
	original := Matrix{
		{{Value: 1}, {Value: 2}},
		{{Value: 3}, {Value: 4}},
	}
	copied := original
	copied[0][1].Value = 9
	return original[0][1].Value, copied[0][1].Value
}

func SparseLiteralZerosAreFresh() (int32, int32, int32) {
	values := [3]Box{1: {Value: 5}}
	values[0].Value = 7
	values[2].Value = 9
	return values[0].Value, values[1].Value, values[2].Value
}

func Equal(left, right Boxes) bool {
	return left == right
}

func PointerStore(value Boxes) Boxes {
	target := &value[1]
	*target = Box{Value: 11}
	return value
}

func CallIsolation(value Boxes) (Boxes, Boxes) {
	copy := Identity(value)
	copy[0].Value = 12
	return value, copy
}

func Identity(value Boxes) Boxes {
	return value
}

func First(value Boxes) int32 {
	return value[0].Value
}

func Second(value Boxes) int32 {
	return value[1].Value
}

func SliceDefinedArrayAliases(value Boxes) bool {
	view := value[:1:2]
	view[0].Value = 21
	expanded := view[:2]
	expanded[1].Value = 22
	return value[0].Value == 21 && value[1].Value == 22
}

func SlicePointerArrayAliases(value *Boxes) bool {
	view := value[1:]
	view[0].Value = 23
	return value[1].Value == 23
}

func SlicePointerArrayAliasesValue(value Boxes) bool {
	return SlicePointerArrayAliases(&value)
}

func SlicePlainArrayAliases(value [2]Box) bool {
	view := value[:]
	view[0].Value = 24
	return value[0].Value == 24
}

func SlicePlainArrayAliasesValue(value Boxes) bool {
	return SlicePlainArrayAliases([2]Box(value))
}

func SliceEvaluationOrder() (int32, int32, int32, int32) {
	array := [4]int32{1, 2, 3, 4}
	events := [4]int32{}
	next := int32(0)
	_ = sliceOperand(&array, &events, &next)[sliceBound(&events, &next, 2, 0):sliceBound(&events, &next, 3, 3):sliceBound(&events, &next, 4, 4)]
	return events[0], events[1], events[2], events[3]
}

func sliceOperand(
	value *[4]int32,
	events *[4]int32,
	next *int32,
) *[4]int32 {
	events[*next] = 1
	*next++
	return value
}

func sliceBound(
	events *[4]int32,
	next *int32,
	marker int32,
	value int32,
) int32 {
	events[*next] = marker
	*next++
	return value
}

func SliceHighPanic() {
	value := [2]int32{}
	high := int32(3)
	_ = value[:high]
}

func SliceMaxPanic() {
	value := [2]int32{}
	maximum := int32(3)
	_ = value[:1:maximum]
}

func SliceLowPanic() {
	value := [2]int32{}
	low := int32(-1)
	_ = value[low:]
}
