package aggregatemap

type unreachableKey struct {
	value uint16
}

func unreachableMap() map[unreachableKey]uint16 {
	return make(map[unreachableKey]uint16)
}
