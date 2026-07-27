package data

func NumberValue(left, right int64) int64 {
	return left/right + left%right
}

func StringByte(value string) byte {
	return value[1]
}

func StringWindow(value string) string {
	return value[1:3]
}

func PointerValue(value int32) int32 {
	pointer := new(int32)
	*pointer = value
	return *pointer
}

func ArrayValue(value int32) int32 {
	values := [2]int32{value, value + 1}
	values[0] = value + 2
	return values[0]*10 + values[1]
}

func SliceValue(value int32) int32 {
	values := []int32{value, value + 1}
	values[1] = value + 2
	return values[0]*10 + values[1]
}

func MapValue(value int32) int32 {
	values := map[string]int32{"left": value}
	values["right"] = value + 1
	return values["left"]*10 + values["right"]
}

func PointerPanic() int32 {
	var pointer *int32
	return *pointer
}

func ArrayPanic(index int) int32 {
	values := [1]int32{1}
	return values[index]
}

func SlicePanic(index int) int32 {
	values := []int32{1}
	return values[index]
}

func StringPanic(value string, index int) byte {
	return value[index]
}

func MapPanic() {
	var values map[string]int32
	values["missing"] = 1
}

func DividePanic(divisor int64) int64 {
	return 1 / divisor
}

var storeTrace int32

func markStore(step int32) {
	storeTrace = storeTrace*10 + step
}

func sliceStoreReceiver(values []int32) []int32 {
	markStore(1)
	return values
}

func sliceStoreIndex() int {
	markStore(2)
	return 0
}

func mapStoreReceiver(values map[string]int32) map[string]int32 {
	markStore(1)
	return values
}

func mapStoreKey() string {
	markStore(2)
	return "key"
}

func storePair() (int32, int32) {
	markStore(3)
	return 7, 8
}

func chooseStoreSecond(first int32, value int32) int32 {
	return value
}

func SliceStoreOrder() int32 {
	storeTrace = 0
	values := []int32{0}
	sliceStoreReceiver(values)[sliceStoreIndex()] = chooseStoreSecond(storePair())
	return storeTrace*10 + values[0]
}

func MapStoreOrder() int32 {
	storeTrace = 0
	values := make(map[string]int32)
	mapStoreReceiver(values)[mapStoreKey()] = chooseStoreSecond(storePair())
	return storeTrace*10 + values["key"]
}
