package slicevalues

type DestinationValues []int32
type SourceValues []int32
type ByteText string

var packageValues = []int32{2, 3}

func Identity(values []int32) []int32 {
	return values
}

func PackageSliceAliasesBacking() int32 {
	alias := packageValues
	alias[0] = 7
	return packageValues[0]
}

func NilIsNil() bool {
	var values []int32
	return values == nil
}

func namedSlice() (values []int32) {
	return
}

func NamedSliceZeroIsNil() bool {
	return namedSlice() == nil
}

func EmptyIsNil() bool {
	values := make([]int32, 0)
	return values == nil
}

func MakeShape() int {
	values := make([]int32, 2, 5)
	return len(values)*10 + cap(values)
}

func LiteralIndex() int32 {
	values := []int32{4, 5, 6}
	return values[1]
}

func KeyedLiteral() int32 {
	values := []int32{2: 5, 7}
	return values[0]*1000 +
		values[1]*100 +
		values[2]*10 +
		values[3]
}

func DescriptorAliasesBacking() int32 {
	values := []int32{1, 2, 3}
	copyOfDescriptor := values
	copyOfDescriptor[1] = 9
	return values[1]
}

func appendParameter(values []int32) {
	values = append(values, 8)
}

func ParameterDescriptorIsIndependent() bool {
	values := make([]int32, 1, 2)
	appendParameter(values)
	expanded := values[:2]
	return len(values) == 1 && expanded[1] == 8
}

func TwoIndexSlice() int {
	values := make([]int32, 3, 6)
	view := values[1:2]
	return len(view)*10 + cap(view)
}

func ThreeIndexSlice() int {
	values := make([]int32, 3, 6)
	view := values[1:2:4]
	return len(view)*10 + cap(view)
}

func AppendReusesBacking() int32 {
	values := make([]int32, 2, 4)
	values[0] = 1
	grown := append(values, 3, 4)
	grown[0] = 9
	expanded := values[:4]
	return expanded[0]*100 + expanded[2]*10 + expanded[3]
}

func AppendReallocates() int32 {
	values := []int32{1, 2}
	grown := append(values, 3)
	grown[0] = 9
	return values[0]
}

func AppendNoValues() int {
	values := make([]int32, 2, 4)
	same := append(values)
	return len(same)*10 + cap(same)
}

func AppendGrowthCapacity() int {
	values := []int32{1, 2}
	values = append(values, 3)
	return cap(values)
}

func AppendReallocationZeroTail() int32 {
	values := []int32{1, 2}
	grown := append(values, 3)
	expanded := grown[:cap(grown)]
	return expanded[3]
}

func AppendSpread() int32 {
	values := []int32{1, 2}
	suffix := []int32{3, 4}
	result := append(values, suffix...)
	return result[3]
}

func AppendSpreadOverlap() int32 {
	values := make([]int32, 3, 6)
	values[0], values[1], values[2] = 1, 2, 3
	result := append(values[:1], values[1:3]...)
	return result[0]*100 + result[1]*10 + result[2]
}

func AppendDistinctNamedSlices() int32 {
	values := DestinationValues{1}
	suffix := SourceValues{2, 3}
	result := append(values, suffix...)
	return result[0]*100 + result[1]*10 + result[2]
}

func AppendStringBytes() int32 {
	result := append([]byte{1}, "é"...)
	return int32(result[0])*10000 +
		int32(result[1])*100 +
		int32(result[2])
}

func AppendDefinedStringBytes() int32 {
	result := append([]byte{1}, ByteText("é")...)
	return int32(result[0])*10000 +
		int32(result[1])*100 +
		int32(result[2])
}

func CopyStringBytes() int32 {
	result := []byte{9, 9, 9, 9}
	count := copy(result[1:], "é")
	return int32(count)*1000000 +
		int32(result[0])*10000 +
		int32(result[1])*100 +
		int32(result[2])
}

func CopyDefinedStringBytes() int32 {
	result := make([]byte, 3)
	count := copy(result, ByteText("é"))
	return int32(count)*10000 +
		int32(result[0])*100 +
		int32(result[1])
}

func AppendLargeSpread() int32 {
	suffix := make([]int32, 200000)
	suffix[len(suffix)-1] = 7
	result := append([]int32{}, suffix...)
	return result[len(result)-1]
}

func IndexUpdates() int32 {
	values := []int32{2}
	values[0] += 3
	values[0] *= 4
	values[0]--
	return values[0]
}

func StringIndexCompound() string {
	values := []string{"a"}
	values[0] += "b"
	return values[0]
}

var storeTrace int32

func markSlice(values []int32, mark int32) []int32 {
	storeTrace = storeTrace*10 + mark
	return values
}

func markIndex(mark int32, index int) int {
	storeTrace = storeTrace*10 + mark
	return index
}

func markValue(mark int32, value int32) int32 {
	storeTrace = storeTrace*10 + mark
	return value
}

func ParallelStoreOrder() int32 {
	storeTrace = 0
	values := []int32{0, 0}
	markSlice(values, 1)[markIndex(2, 0)],
		markSlice(values, 3)[markIndex(4, 1)] =
		markValue(5, 7), markValue(6, 8)
	return storeTrace*100 + values[0]*10 + values[1]
}

func CopyOverlapping() int32 {
	values := []int32{1, 2, 3, 4}
	copy(values[1:], values)
	return values[0]*1000 +
		values[1]*100 +
		values[2]*10 +
		values[3]
}

func CopyDistinct() int32 {
	source := []int32{7, 8, 9}
	target := make([]int32, 2)
	copy(target, source)
	return target[0]*10 + target[1]
}

func CopyCount() int {
	source := []int32{7, 8, 9}
	target := make([]int32, 2)
	return copy(target, source)
}

func NilSliceStaysNil() bool {
	var values []int32
	view := values[:]
	return view == nil
}

func BoolElements() bool {
	values := make([]bool, 2)
	values[1] = true
	return values[0] || values[1]
}

func ShadowedLenIsOrdinaryCall() int32 {
	len := func(value int32) int32 {
		return value + 1
	}
	return len(4)
}

func IndexBoundsPanic() int32 {
	values := []int32{1}
	return values[1]
}

func StoreBoundsPanic() {
	values := []int32{1}
	values[1] = 2
}

func SliceBoundsPanic() int {
	values := []int32{1}
	return len(values[:2])
}

func NegativeHighBoundsPanic() int {
	values := []int32{1}
	high := -1
	return len(values[:high])
}
