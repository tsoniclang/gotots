package aggregateslice

type Box struct {
	Value int32
}

type Pair [2]Box

func MakeZerosAreFresh() bool {
	values := make([]Box, 2, 4)
	values[0].Value = 7
	return values[1].Value == 0
}

func CapacityZerosAreFresh() bool {
	values := make([]Box, 1, 3)
	expanded := values[:3]
	expanded[1].Value = 7
	return expanded[2].Value == 0
}

func LiteralCopiesValues() bool {
	value := Box{Value: 3}
	values := []Box{value, value}
	values[0].Value = 7
	return value.Value == 3 && values[1].Value == 3
}

func SparseLiteralZerosAreFresh() bool {
	values := []Box{2: {Value: 3}}
	values[0].Value = 7
	return values[1].Value == 0 && values[2].Value == 3
}

func AppendReuseAliasesBackingAndCopiesArgument() bool {
	values := make([]Box, 1, 3)
	values[0].Value = 1
	appended := Box{Value: 2}
	grown := append(values, appended)
	appended.Value = 8
	grown[0].Value = 9
	grown[1].Value = 7
	expanded := values[:2]
	return values[0].Value == 9 &&
		expanded[1].Value == 7 &&
		appended.Value == 8
}

func AppendReallocationCopiesExisting() bool {
	values := []Box{{Value: 1}}
	grown := append(values, Box{Value: 2})
	grown[0].Value = 9
	return values[0].Value == 1 && grown[1].Value == 2
}

func AppendTailZerosAreFresh() bool {
	values := make([]Box, 1, 4)
	values[0].Value = 1
	grown := append(values, Box{Value: 2})
	expanded := grown[:cap(grown)]
	expanded[2].Value = 7
	return expanded[3].Value == 0
}

func AppendSpreadCopiesValues() bool {
	values := []Box{{Value: 1}}
	appended := []Box{{Value: 2}}
	grown := append(values, appended...)
	appended[0].Value = 8
	grown[0].Value = 9
	grown[1].Value = 7
	return values[0].Value == 9 && appended[0].Value == 8 && grown[1].Value == 7
}

func AppendSpreadOverlapSnapshotsValues() bool {
	values := make([]Box, 3, 6)
	values[0].Value = 1
	values[1].Value = 2
	values[2].Value = 3
	grown := append(values[:1], values[1:3]...)
	grown[1].Value = 9
	return grown[0].Value == 1 &&
		grown[1].Value == 9 &&
		grown[2].Value == 3 &&
		values[1].Value == 9 &&
		values[2].Value == 3
}

func CopyDistinctCopiesValues() bool {
	source := []Box{{Value: 1}, {Value: 2}}
	target := make([]Box, 2)
	copy(target, source)
	target[0].Value = 9
	return source[0].Value == 1 && target[1].Value == 2
}

func CopyOverlapSnapshotsValues() bool {
	values := []Box{{Value: 1}, {Value: 2}, {Value: 3}}
	copy(values[1:], values)
	values[1].Value = 9
	return values[0].Value == 1 &&
		values[1].Value == 9 &&
		values[2].Value == 2
}

func AddressTargetsBackingElement() bool {
	values := []Box{{Value: 1}}
	target := &values[0]
	target.Value = 7
	return values[0].Value == 7
}

func ArrayElementsCopyOnAppend() bool {
	value := Pair{{Value: 1}, {Value: 2}}
	values := append([]Pair{}, value)
	values[0][0].Value = 9
	return value[0].Value == 1 && values[0][1].Value == 2
}

func ElidedNestedLiterals() bool {
	slices := [][]Box{
		{{Value: 1}},
		{{Value: 2}},
	}
	maps := []map[int32]Box{
		{1: {Value: 3}},
	}
	slices[0][0].Value = 9
	copied := maps[0][1]
	copied.Value = 8
	return slices[1][0].Value == 2 && maps[0][1].Value == 3
}
