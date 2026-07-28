package slicevalues

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
