package clearvalues

type Box struct {
	Value int32
}

func ClearGeneric[C ~[]Box | ~map[int32]Box](values C) {
	clear(values)
}

func ClearScalarSlice() int32 {
	values := []int32{1, 2, 3}
	clear(values)
	return values[1]
}

func ClearAggregateSlice() int32 {
	values := []Box{{Value: 1}, {Value: 2}, {Value: 3}}
	clear(values)
	return values[1].Value
}

func ClearScalarMap() int32 {
	values := map[int32]int32{1: 2, 2: 3}
	clear(values)
	return int32(len(values))
}

func ClearAggregateMap() int32 {
	values := map[int32]Box{1: {Value: 2}}
	clear(values)
	return values[1].Value
}

func ClearGenericSlice() int32 {
	values := []Box{{Value: 1}}
	ClearGeneric(values)
	return values[0].Value
}

func ClearGenericMap() int32 {
	values := map[int32]Box{1: {Value: 2}}
	ClearGeneric(values)
	return values[1].Value
}

func ClearNilValues() int32 {
	var values []int32
	var mapping map[int32]int32
	clear(values)
	clear(mapping)
	return int32(len(values) + len(mapping))
}
