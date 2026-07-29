package wave7generics

import "example.com/wave7generics/support"

type Integer interface {
	~int32 | ~int64
}

type ShiftCount interface {
	~uint8 | ~uint16
}

type Signed interface {
	~int32 | ~int64
}

type Unsigned interface {
	~uint32 | ~uint64
}

type Boolean interface {
	~bool
}

type Floating interface {
	~float32 | ~float64
}

type ComplexNumber interface {
	~complex64 | ~complex128
}

type Text interface {
	~string
}

type Sequence[E any] interface {
	~[]E
}

type Pair[E any] interface {
	~[2]E
}

type ByteSequence interface {
	~string | ~[]byte
}

type ValueReader interface {
	ReadValue() int32
}

type Box[T any] struct {
	Value T
}

type Alias[T any] = Box[T]

type Node[T any] struct {
	Value T
	Next  *Node[T]
}

type Left[T any] struct {
	Value T
	Right *Right[T]
}

type Right[T any] struct {
	Value T
	Left  *Left[T]
}

type ComparableBox[T comparable] struct {
	Value T
}

type ReaderValue struct {
	Value int32
}

func (value ReaderValue) ReadValue() int32 {
	return value.Value
}

func Identity[T any](value T) T {
	return value
}

func Zero[T any]() T {
	var result T
	return result
}

func Add[T Integer](left, right T) T {
	return left + right
}

func Twice[T Integer](value T) T {
	return Add(value, value)
}

func Shift[T Integer, U ShiftCount](value T, count U) T {
	return value << count
}

func Equal[T comparable](left, right T) bool {
	return left == right
}

func NewBox[T any](value T) Box[T] {
	return Box[T]{Value: value}
}

func (box Box[T]) Get() T {
	return box.Value
}

func (box ComparableBox[T]) Same(other ComparableBox[T]) bool {
	return box.Value == other.Value
}

func AliasBox[T any](value T) Alias[T] {
	return Alias[T]{Value: value}
}

func NewNode[T any](value T) Node[T] {
	return Node[T]{Value: value}
}

func RecursiveValue() int32 {
	tail := NewNode(int32(2))
	head := NewNode(int32(1))
	head.Next = &tail
	return head.Next.Value
}

func MutualValue() int32 {
	left := Left[int32]{Value: 3}
	right := Right[int32]{Value: 4, Left: &left}
	left.Right = &right
	return left.Right.Value + right.Left.Value
}

func ZeroBox[T any]() Box[T] {
	var result Box[T]
	return result
}

func CopyBox[T any](value Box[T]) Box[T] {
	return value
}

func EqualBox[T comparable](left, right Box[T]) bool {
	return left == right
}

func Negate[T Signed](value T) T {
	return -value
}

func Positive[T Signed](value T) T {
	return +value
}

func Complement[T Unsigned](value T) T {
	return ^value
}

func LogicalNot[T Boolean](value T) T {
	return !value
}

func FloatArithmetic[T Floating](left, right T) T {
	return -(+left + right) * right / right
}

func ComplexArithmetic[T ComplexNumber](left, right T) T {
	return -(left + right*left/right)
}

func TextOperations[T Text](left, right T) (T, bool) {
	return left + right, left < right
}

func Arithmetic[T Integer](left, right T) T {
	return (left-right)*right/right%left | left&right ^ left&^right
}

func AuditBigIntOperations() int32 {
	return Arithmetic(int32(6), int32(3))
}

func Ordered[T Integer](left, right T) bool {
	return left < right &&
		left <= right &&
		right > left &&
		right >= left &&
		left != right
}

func ConvertValue[T Signed, U Signed](value T) U {
	return U(value)
}

func SequenceValue[S Sequence[E], E any](values S, index int) E {
	return values[index]
}

func SequenceSize[S Sequence[E], E any](values S) int32 {
	return int32(len(values) + cap(values))
}

func PairValue[P Pair[E], E any](values P, index int) E {
	return values[index]
}

func ByteSequenceValue[S ByteSequence](values S, index int) byte {
	return values[index]
}

func ByteSequenceSize[S ByteSequence](values S) int32 {
	return int32(len(values))
}

func GenericMapValue[T comparable](key T) int32 {
	values := map[T]int32{key: 12}
	return values[key]
}

func GenericMapOperations[K comparable, V any](
	key K,
	value V,
) (V, bool, int32) {
	values := make(map[K]V, 1)
	values[key] = value
	result, present := values[key]
	count := int32(0)
	for current := range values {
		if current == key {
			count++
		}
	}
	delete(values, key)
	values[key] = value
	clear(values)
	return result, present, count + int32(len(values))
}

func ReadConstraint[T ValueReader](value T) int32 {
	return value.ReadValue()
}

func InterfaceValue[T any](value T) any {
	return value
}

func ConstructedValues[T any](value T) ([]T, *T, [2]T) {
	items := []T{value}
	pointer := &value
	array := [2]T{value, value}
	return items, pointer, array
}

func ZeroIterator(yield func() bool) {
	for index := int32(0); index < 4; index++ {
		if !yield() {
			return
		}
	}
}

func OneIterator(yield func(int32) bool) {
	for value := int32(1); value <= 4; value++ {
		if !yield(value) {
			return
		}
	}
}

func TwoIterator(yield func(int32, string) bool) {
	for key := int32(1); key <= 3; key++ {
		if !yield(key, "x") {
			return
		}
	}
}

func SelectOneIterator(counter *int32) func(func(int32) bool) {
	*counter++
	return OneIterator
}

func GenericIteratorSum[T Integer](
	iterator func(func(T) bool),
) T {
	var result T
	for value := range iterator {
		result += value
	}
	return result
}

func BoxIterator[T any](value Box[T]) func(func(Box[T]) bool) {
	return func(yield func(Box[T]) bool) {
		yield(value)
	}
}

func GenericIteratorCopy[T any](value T) T {
	original := Box[T]{Value: value}
	for current := range BoxIterator(original) {
		var zero T
		current.Value = zero
	}
	return original.Value
}

func BadIterator(yield func(int32) bool) {
	if !yield(1) {
		yield(2)
	}
}

func BreaksBadIterator() {
	for range BadIterator {
		break
	}
}

func CallsYieldAfterExit() {
	var saved func(int32) bool
	iterator := func(yield func(int32) bool) {
		saved = yield
	}
	for range iterator {
	}
	saved(1)
}

func RangesNilIterator(iterator func(func() bool)) {
	for range iterator {
	}
}

func IteratorReturnBoundary() int32 {
	for range ZeroIterator {
		return 1
	}
	return 0
}

func IteratorLabelBoundary() {
iterator:
	for range ZeroIterator {
		continue iterator
	}
}

func AuditIteratorRanges() []int32 {
	zeroCount := int32(0)
	for range ZeroIterator {
		zeroCount++
		if zeroCount == 1 {
			continue
		}
		break
	}

	evaluations := int32(0)
	oneSum := int32(0)
	for value := range SelectOneIterator(&evaluations) {
		if value == 2 {
			continue
		}
		if value == 4 {
			break
		}
		oneSum += value
	}

	twoSum := int32(0)
	for key, value := range TwoIterator {
		twoSum += key + int32(len(value))
	}

	assigned := int32(0)
	for assigned = range OneIterator {
		break
	}

	return []int32{
		zeroCount,
		evaluations,
		oneSum,
		twoSum,
		assigned,
		GenericIteratorSum[int32](OneIterator),
		GenericIteratorCopy(int32(23)),
	}
}

func RecursiveAdd[T Integer](value, increment T, remaining int32) T {
	if remaining == 0 {
		return value
	}
	return RecursiveAdd(value+increment, increment, remaining-1)
}

func MutualAddA[T Integer](value, increment T, remaining int32) T {
	if remaining == 0 {
		return value
	}
	return MutualAddB(value+increment, increment, remaining-1)
}

func MutualAddB[T Integer](value, increment T, remaining int32) T {
	if remaining == 0 {
		return value
	}
	return MutualAddA(value+increment, increment, remaining-1)
}

func CallableValues() []int32 {
	identity := Identity[int32]
	box := Box[int32]{Value: 8}
	boundGet := box.Get
	unboundGet := Box[int32].Get
	externalMake := support.Make[int32]
	external := externalMake(9)
	externalGet := external.Get
	return []int32{
		identity(7),
		boundGet(),
		unboundGet(box),
		externalGet(),
	}
}

func AuditGenericMethodAdapters() []int32 {
	first := ComparableBox[int32]{Value: 17}
	equal := ComparableBox[int32]{Value: 17}
	different := ComparableBox[int32]{Value: 18}
	methodValue := first.Same
	methodExpression := ComparableBox[int32].Same
	if !methodValue(equal) ||
		methodValue(different) ||
		!methodExpression(first, equal) ||
		methodExpression(first, different) {
		return []int32{-1}
	}
	return []int32{first.Value, equal.Value, different.Value}
}

func AuditFunctions() []int32 {
	first := Identity(int32(4))
	second := Add[int32](first, 5)
	doubled := Twice(int32(3))
	zero := Zero[int32]()
	if !Equal(second, int32(9)) {
		return []int32{-1}
	}
	return []int32{second, doubled, zero}
}

func Audit() []int32 {
	first := Identity(int32(4))
	second := Add[int32](first, 5)
	box := NewBox(second)
	alias := AliasBox(second)
	copied := CopyBox(box)
	empty := ZeroBox[int32]()
	firstComparable := ComparableBox[int32]{Value: 5}
	secondComparable := ComparableBox[int32]{Value: 5}
	boundSame := firstComparable.Same
	unboundSame := ComparableBox[int32].Same
	external := support.Make(int32(6))
	constructedSlice, constructedPointer, constructedArray :=
		ConstructedValues(int32(11))
	mapValue, mapPresent, mapCount :=
		GenericMapOperations(int32(15), int32(17))
	textValue, textOrdered := TextOperations("go", "ts")
	interfaceValue := InterfaceValue(int32(10))
	zero := Zero[int32]()
	if !Equal(copied.Get(), int32(9)) ||
		!Equal(alias.Get(), int32(9)) ||
		!Equal(empty.Get(), int32(0)) ||
		!EqualBox(box, copied) ||
		!firstComparable.Same(secondComparable) ||
		!boundSame(secondComparable) ||
		!unboundSame(firstComparable, secondComparable) ||
		!mapPresent ||
		!textOrdered ||
		textValue != "gots" ||
		FloatArithmetic(float64(6), float64(3)) != float64(-9) ||
		ComplexArithmetic(complex128(2+3i), complex128(1-1i)) !=
			complex128(-4-2i) ||
		interfaceValue != int32(10) ||
		!Equal(external.Get(), int32(6)) {
		return []int32{-1}
	}
	return []int32{
		box.Get(),
		zero,
		RecursiveValue(),
		MutualValue(),
		external.Get(),
		RecursiveAdd(int32(1), int32(1), 2),
		MutualAddA(int32(1), int32(1), 2),
		CallableValues()[0],
		CallableValues()[1],
		CallableValues()[2],
		CallableValues()[3],
		Positive(int32(2)),
		Negate(int32(2)),
		int32(Complement(uint32(1))),
		boolToInt32(LogicalNot(false)),
		Shift(int32(1), uint8(3)),
		boolToInt32(Ordered(int32(2), int32(3))),
		int32(ConvertValue[int32, int64](int32(13))),
		SequenceValue([]int32{14}, 0),
		SequenceSize(make([]int32, 1, 2)),
		PairValue([2]int32{18, 19}, 1),
		int32(ByteSequenceValue("A", 0)),
		int32(ByteSequenceValue([]byte{66}, 0)),
		ByteSequenceSize("abc"),
		ByteSequenceSize([]byte{1, 2}),
		GenericMapValue(int32(15)),
		mapValue,
		mapCount,
		ReadConstraint(ReaderValue{Value: 16}),
		constructedSlice[0],
		*constructedPointer,
		constructedArray[1],
	}
}

func boolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
