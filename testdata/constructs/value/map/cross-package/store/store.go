package store

func New(value int32) map[int32]int32 {
	return map[int32]int32{1: value}
}

func Identity(values map[int32]int32) map[int32]int32 {
	return values
}
