package arrayvalues

var PackageZero [2]int32
var PackageLiteral = [3]int32{1, 2, 3}
var trace int32
var storeTrace int32
var readTrace int32

type Holder struct {
	Values [2]int32
}

func ZeroIsFresh() bool {
	var left [2]int32
	right := left
	right[0] = 7
	return left[0] == 0 && right[0] == 7
}

func CopyIsValue() bool {
	source := [3]int32{1, 2, 3}
	copy := source
	copy[1] = 9
	return source[1] == 2 && copy[1] == 9
}

func EqualValues() bool {
	left := [3]int32{1, 2, 3}
	right := [3]int32{0: 1, 2: 3, 1: 2}
	return left == right
}

func NotEqualValues() bool {
	left := [3]int32{1, 2, 3}
	right := [3]int32{1, 2, 4}
	return left != right
}

func LiteralValues() int32 {
	positional := [4]int32{1, 2}
	keyed := [4]int32{3: 7, 1: 5}
	return positional[0] + positional[2] + keyed[1] + keyed[3]
}

func BoolValues() bool {
	values := [3]bool{true, 2: true}
	return values[0] && !values[1] && values[2]
}

func IndexStore(index int, value int32) int32 {
	values := [3]int32{1, 2, 3}
	values[index] = value
	return values[index]
}

func LengthAndCapacity() int {
	values := [5]int32{}
	return len(values) + cap(values)
}

func InferredLength() int {
	values := [...]int32{3: 7}
	return len(values) + cap(values)
}

func copyArgument(value [2]int32) [2]int32 {
	value[0] = 8
	return value
}

func ArgumentAndResultCopy() bool {
	source := [2]int32{1, 2}
	result := copyArgument(source)
	result[1] = 9
	return source[0] == 1 && source[1] == 2 &&
		result[0] == 8 && result[1] == 9
}

func ZeroLength() bool {
	var left [0]int32
	var right [0]int32
	return left == right && len(left) == 0 && cap(right) == 0
}

func PackageValuesAreIsolated() bool {
	copy := PackageLiteral
	copy[0] = 9
	return PackageZero[0] == 0 &&
		PackageLiteral[0] == 1 &&
		copy[0] == 9
}

func PackageIndexStore() bool {
	PackageZero[1] = 6
	return PackageZero[0] == 0 && PackageZero[1] == 6
}

func Two() [2]int32 {
	return [2]int32{1, 2}
}

func Three() [3]int32 {
	return [3]int32{1, 2, 3}
}

func AcceptTwo(value [2]int32) int32 {
	return value[0] + value[1]
}

func StructFieldCopyAndEquality() bool {
	original := Holder{Values: [2]int32{1, 2}}
	copy := original
	copy.Values[0] = 9
	return original.Values[0] == 1 &&
		copy.Values[0] == 9 &&
		original != copy
}

func mark(value int32) int32 {
	trace = trace*10 + value
	return value
}

func KeyedEvaluationOrder() int32 {
	trace = 0
	values := [4]int32{1: mark(1), mark(2), 0: mark(3)}
	return trace*100 + values[0]*10 + values[2]
}

func GoArray() int32 {
	return 4
}

func RuntimeNameCollision() int32 {
	values := [1]int32{3}
	return GoArray() + values[0]
}

func nextIndex() int {
	storeTrace = storeTrace*10 + 1
	return 0
}

func resultPair() (int32, int32) {
	storeTrace = storeTrace*10 + 2
	return 7, 8
}

func chooseSecond(first int32, value int32) int32 {
	return value
}

func StoreEvaluationOrder() int32 {
	storeTrace = 0
	values := [1]int32{}
	values[nextIndex()] = chooseSecond(resultPair())
	return storeTrace*10 + values[0]
}

func readArray() [1]int32 {
	readTrace = readTrace*10 + 1
	return [1]int32{7}
}

func indexPair() (int, int) {
	readTrace = readTrace*10 + 2
	return 0, 0
}

func chooseIndex(first int, second int) int {
	return first + second
}

func ReadEvaluationOrder() int32 {
	readTrace = 0
	value := readArray()[chooseIndex(indexPair())]
	return readTrace*10 + value
}
