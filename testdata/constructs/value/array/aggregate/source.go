package aggregatearray

type Box struct {
	Value int32
}

type Boxes [2]Box
type Matrix [2][2]Box

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
