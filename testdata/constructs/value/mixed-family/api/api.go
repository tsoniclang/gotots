package api

import "example.com/mixedfamily/data"

func NumberValue(left, right int64) int64 {
	return data.NumberValue(left, right)
}

func StringByte(value string) byte {
	return data.StringByte(value)
}

func StringWindow(value string) string {
	return data.StringWindow(value)
}

func PointerValue(value int32) int32 {
	return data.PointerValue(value)
}

func ArrayValue(value int32) int32 {
	return data.ArrayValue(value)
}

func SliceValue(value int32) int32 {
	return data.SliceValue(value)
}

func MapValue(value int32) int32 {
	return data.MapValue(value)
}

func PointerPanic() int32 {
	return data.PointerPanic()
}

func ArrayPanic(index int) int32 {
	return data.ArrayPanic(index)
}

func SlicePanic(index int) int32 {
	return data.SlicePanic(index)
}

func StringPanic(value string, index int) byte {
	return data.StringPanic(value, index)
}

func MapPanic() {
	data.MapPanic()
}

func DividePanic(divisor int64) int64 {
	return data.DividePanic(divisor)
}

func SliceStoreOrder() int32 {
	return data.SliceStoreOrder()
}

func MapStoreOrder() int32 {
	return data.MapStoreOrder()
}
