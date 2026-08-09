package conversion

func NarrowSigned(value int64) int8 {
	return int8(value)
}

func NarrowUnsigned(value int64) uint8 {
	return uint8(value)
}

func Sign32(value int32) uint32 {
	return uint32(value)
}

func Sign64(value int64) uint64 {
	return uint64(value)
}

func BackSign64(value uint64) int64 {
	return int64(value)
}

var EvaluationCount int32

func nextInteger() int64 {
	EvaluationCount++
	return -1
}

func Sign64EvaluatesOnce() int32 {
	EvaluationCount = 0
	converted := uint64(nextInteger())
	if converted == 0 {
		return -1
	}
	return EvaluationCount
}

func Widen(value int8) int64 {
	return int64(value)
}

func IntegerToFloat64(value int64) float64 {
	return float64(value)
}

func UnsignedToFloat32(value uint64) float32 {
	return float32(value)
}

func FloatToInt8(value float64) int8 {
	return int8(value)
}

func FloatToUint32(value float64) uint32 {
	return uint32(value)
}

func FloatToInt64(value float64) int64 {
	return int64(value)
}

func FloatToUint64(value float64) uint64 {
	return uint64(value)
}

func WidenFloat(value float32) float64 {
	return float64(value)
}

func NarrowFloat(value float64) float32 {
	return float32(value)
}

func WidenComplex(realPart, imaginaryPart float32) (float64, float64) {
	value := complex128(complex(realPart, imaginaryPart))
	return real(value), imag(value)
}

func NarrowComplex(realPart, imaginaryPart float64) (float32, float32) {
	value := complex64(complex(realPart, imaginaryPart))
	return real(value), imag(value)
}

func nextComplex() complex64 {
	EvaluationCount++
	return 1 + 2i
}

func ComplexEvaluatesOnce() int32 {
	EvaluationCount = 0
	converted := complex128(nextComplex())
	if real(converted) == 0 {
		return -1
	}
	return EvaluationCount
}

func ConstantInteger() int16 {
	return int16(255)
}

func ConstantFloat() float32 {
	return float32(16777217)
}

func ConstantComplex() complex64 {
	return complex64(1.5 - 2.25i)
}

type Bytes []byte
type Text string

func IntegerStringSummary() int32 {
	text := string(int32(0x1f600))
	return int32(len(text))*100000000 +
		int32(text[0])*1000000 +
		int32(text[1])*10000 +
		int32(text[2])*100 +
		int32(text[3])
}

func BytesStringSummary() int32 {
	text := string([]byte{0xff, 'A'})
	return int32(len(text))*100000 +
		int32(text[0])*100 +
		int32(text[1])
}

func String(value string) string {
	return value
}

func ShadowedStringIntrinsic() string {
	return string([]byte{'A'}) + String("B")
}

func RunesStringSummary() int32 {
	text := string([]rune{'A', 'é', '😀'})
	return int32(len(text))*100000000 +
		int32(text[0])*1000000 +
		int32(text[1])*10000 +
		int32(text[2])*100 +
		int32(text[3])
}

func StringBytesSummary() int32 {
	values := []byte("Aé")
	return int32(len(values))*1000000 +
		int32(values[0])*10000 +
		int32(values[1])*100 +
		int32(values[2])
}

func StringRunesSummary() int32 {
	values := []rune("Aé😀")
	return int32(len(values))*100000000 +
		values[0]*1000000 +
		values[1]*1000 +
		values[2]
}

func InvalidStringRunesSummary() int64 {
	values := []rune("\xffA\xc0")
	return int64(values[0])*1000000 +
		int64(values[1])*1000 +
		int64(values[2])
}

func InvalidStringBoundarySummary() int64 {
	values := []rune("\xe2\x82\xf0\x9f\x98\xed\xa0\x80\xf4\x90\x80\x80")
	return int64(len(values))*100000 + int64(values[0])
}

func InvalidRuneStringSummary() int32 {
	text := string(int32(0xd800))
	return int32(len(text))*1000000 +
		int32(text[0])*10000 +
		int32(text[1])*100 +
		int32(text[2])
}

func NilSlicesToString() bool {
	var bytes []byte
	var runes []rune
	return string(bytes) == "" && string(runes) == ""
}

func DefinedStringConversions() int32 {
	values := Bytes(Text("é"))
	text := Text(values)
	return int32(len(values))*1000000 +
		int32(values[0])*10000 +
		int32(values[1])*100 +
		int32(len(text))
}

type TaggedLeft struct {
	Value int32
	Pair  [2]int32 `json:"left"`
}

type TaggedRight struct {
	Value int32
	Pair  [2]int32 `json:"right"`
}

func StructConversionCopies() int32 {
	left := TaggedLeft{Value: 3, Pair: [2]int32{4, 5}}
	right := TaggedRight(left)
	right.Pair[0] = 9
	return left.Pair[0]*100 + right.Value*10 + right.Pair[0]
}

func StructConversionReusesDefinition() int32 {
	first := TaggedRight(TaggedLeft{Value: 2})
	second := TaggedRight(TaggedLeft{Value: 5})
	return first.Value*10 + second.Value
}

func AnonymousStructConversion() int32 {
	left := struct {
		Value int32 `json:"left"`
	}{Value: 7}
	right := struct {
		Value int32 `json:"right"`
	}(left)
	return right.Value
}

type Pair [2]int32
type Numbers []int32

func SliceToArrayCopies() int32 {
	values := []int32{1, 2, 3}
	pair := [2]int32(values)
	pair[0] = 9
	return values[0]*10 + pair[0]
}

func DefinedSliceToArray() int32 {
	values := Numbers{3, 4}
	pair := Pair(values)
	return pair[0]*10 + pair[1]
}

func AggregateSliceToArrayCopies() int32 {
	values := []TaggedLeft{{Value: 4}, {Value: 5}}
	pair := [2]TaggedLeft(values)
	pair[0].Value = 9
	return values[0].Value*10 + pair[0].Value
}

func shortSlice() []int32 {
	EvaluationCount++
	return []int32{1, 2}
}

func SliceToArrayPanics() {
	EvaluationCount = 0
	_ = [3]int32(shortSlice())
}

func SliceToArrayPanicCount() int32 {
	return EvaluationCount
}

func SliceToArrayPointerAliases() int32 {
	values := []int32{1, 2, 3}
	pointer := (*[2]int32)(values)
	pointer[0] = 7
	values[1] = 8
	before := pointer[0]*10 + pointer[1]
	*pointer = [2]int32{9, 6}
	after := values[0]*10 + values[1]
	return before*100 + after
}

func DefinedSliceToArrayPointerAliases() int32 {
	values := Numbers{3, 4}
	pointer := (*Pair)(values)
	pointer[0] = 5
	values[1] = 6
	*pointer = Pair{8, 9}
	return values[0]*10 + values[1]
}

func AggregateSliceToArrayPointerCopies() int32 {
	values := []TaggedLeft{{Value: 1}, {Value: 2}}
	pointer := (*[2]TaggedLeft)(values)
	replacement := [2]TaggedLeft{{Value: 7}, {Value: 8}}
	*pointer = replacement
	replacement[0].Value = 9
	return values[0].Value*10 + values[1].Value
}

func SliceToArrayPointerIdentity() bool {
	values := []int32{1, 2, 3}
	first := (*[2]int32)(values)
	same := (*[2]int32)(values)
	different := (*[2]int32)(values[1:])
	return first == same && first != different
}

func ZeroLengthSliceToArrayPointers() bool {
	var nilValues []int32
	emptyValues := []int32{}
	return (*[0]int32)(nilValues) == nil &&
		(*[0]int32)(emptyValues) != nil
}

func SliceToArrayPointerPanics() {
	EvaluationCount = 0
	_ = (*[3]int32)(shortSlice())
}

type PointerLeftInt int32
type PointerRightInt int32

func PointerScalarConversion() int32 {
	value := PointerLeftInt(3)
	left := &value
	right := (*PointerRightInt)(left)
	*right = 8
	return int32(value)
}

type PointerLeft struct {
	Value int32
	Pair  [2]int32 `json:"left"`
}

type PointerRight struct {
	Value int32
	Pair  [2]int32 `json:"right"`
}

func PointerStructConversion() int32 {
	value := PointerLeft{Value: 2, Pair: [2]int32{3, 4}}
	left := &value
	right := (*PointerRight)(left)
	right.Value = 7
	right.Pair[0] = 9
	return value.Value*100 + value.Pair[0]*10 + value.Pair[1]
}

func PointerRoundTripIdentity() bool {
	value := PointerLeftInt(1)
	left := &value
	return (*PointerLeftInt)((*PointerRightInt)(left)) == left
}

type PointerNestedLeft struct {
	Value *int32 `json:"left"`
}

type PointerNestedRight struct {
	Value *int32 `json:"right"`
}

func PointerNestedFieldConversion() int32 {
	number := int32(3)
	value := PointerNestedLeft{Value: &number}
	right := (*PointerNestedRight)(&value)
	*right.Value = 8
	return *value.Value
}

type NilCallable func()

func NilConversions() bool {
	pointer := (*int32)(nil)
	slice := []int32(nil)
	mapping := map[int32]int32(nil)
	callable := NilCallable(nil)
	dynamic := any(nil)
	return pointer == nil &&
		slice == nil &&
		mapping == nil &&
		callable == nil &&
		dynamic == nil
}

func GenericNilPointer[T any]() *T {
	return (*T)(nil)
}

func GenericNilPointerIsNil() bool {
	return GenericNilPointer[int32]() == nil
}
