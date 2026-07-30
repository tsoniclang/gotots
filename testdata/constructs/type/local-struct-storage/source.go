package localstructstorage

func Audit() int32 {
	type record struct {
		Value int32
	}
	value := &record{Value: 42}
	return value.Value
}
