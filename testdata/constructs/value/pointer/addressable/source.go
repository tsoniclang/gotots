package addressablepointer

type Box struct {
	Count int32
}

type Holder struct {
	Pointer *Box
}

type Container struct {
	Box Box
}

type PackageBox struct {
	Count int32
}

type Count int32

var Shared = Box{Count: 1}
var WholeShared = PackageBox{Count: 1}
var InitValue int32

func init() {
	value := int32(5)
	pointer := &value
	*pointer++
	InitValue = value
}

func Initialized() int32 {
	return InitValue
}

func Local(value int32) (int32, bool) {
	local := value
	pointer := &local
	*pointer++
	return local, pointer == &local
}

func Escaped(value int32) *int32 {
	return &value
}

func EscapedValue(value int32) int32 {
	pointer := Escaped(value)
	*pointer++
	return *pointer
}

func Parameter(value int32) int32 {
	pointer := &value
	*pointer += 2
	return value
}

func NamedResult(value int32) (result int32) {
	pointer := &result
	*pointer = value
	return
}

func namedArray(value int32) (result [1]int32, pointer *int32) {
	pointer = &result[0]
	result[0] = value
	return
}

func NamedAggregate(value int32) (int32, int32) {
	result, pointer := namedArray(value)
	*pointer++
	return result[0], *pointer
}

func Pair(value int32) (int32, int32) {
	return value, value + 1
}

func Parallel(value int32) (int32, int32) {
	left, right := value, value+1
	pointer := &left
	*pointer++
	return left, right
}

func MultipleResult(value int32) (int32, int32) {
	left, right := Pair(value)
	pointer := &right
	*pointer++
	return left, right
}

func Closure(value int32) func() int32 {
	pointer := &value
	return func() int32 {
		*pointer++
		return value
	}
}

func Shadowed(value int32) (int32, int32) {
	pointer := &value
	inner := func(value int32) int32 {
		innerPointer := &value
		*innerPointer++
		return value
	}
	*pointer++
	return value, inner(value + 10)
}

func Field(value int32) (int32, bool) {
	box := Box{Count: value}
	pointer := &box.Count
	box = Box{Count: value + 2}
	*pointer++
	return box.Count, pointer == &box.Count
}

func NestedField(value int32) (int32, bool) {
	container := Container{Box: Box{Count: value}}
	pointer := &container.Box.Count
	container = Container{Box: Box{Count: value + 2}}
	*pointer++
	return container.Box.Count, pointer == &container.Box.Count
}

func Array(value int32) (int32, bool) {
	values := [2]int32{value, value + 1}
	pointer := &values[1]
	values = [2]int32{value + 2, value + 3}
	*pointer++
	return values[1], pointer == &values[1]
}

func ArrayThroughPointer(value int32) (int32, bool) {
	values := &[2]int32{value, value + 1}
	pointer := &values[1]
	*pointer++
	return *pointer, pointer == &values[1]
}

func ArrayAddress(index int) *int32 {
	values := [1]int32{1}
	return &values[index]
}

func Slice(value int32) (int32, bool, bool) {
	values := []int32{value, value + 1}
	alias := values[:]
	first := &values[1]
	second := &alias[1]
	different := &alias[0]
	*first++
	return alias[1], first == second, first != different
}

func SliceAddress(index int) *int32 {
	values := []int32{1}
	return &values[index]
}

func DefinedArrayAddress(value int32) (int32, bool) {
	values := [1]Count{Count(value)}
	pointer := &values[0]
	*pointer++
	return int32(values[0]), pointer == &values[0]
}

func DefinedSliceAddress(value int32) (int32, bool) {
	values := []Count{Count(value)}
	alias := values[:]
	pointer := &values[0]
	*pointer++
	return int32(alias[0]), pointer == &alias[0]
}

func StructArrayAddress(value int32) (int32, bool) {
	values := [1]Box{{Count: value}}
	pointer := &values[0]
	*pointer = Box{Count: value + 1}
	pointer.Count++
	return values[0].Count, pointer == &values[0]
}

func StructSliceAddress(value int32) (int32, bool) {
	values := []Box{{Count: value}}
	alias := values[:]
	pointer := &values[0]
	*pointer = Box{Count: value + 1}
	pointer.Count++
	return alias[0].Count, pointer == &alias[0]
}

func SliceReallocation(value int32) (bool, int32, int32) {
	values := []int32{value}
	before := &values[0]
	values = append(values, value+1)
	after := &values[0]
	*before++
	return before != after, *before, values[0]
}

func Package(value int32) (int32, bool) {
	pointer := &Shared.Count
	Shared = Box{Count: value}
	*pointer++
	return Shared.Count, pointer == &Shared.Count
}

func PackageValueAddress(value int32) (int32, bool) {
	pointer := &WholeShared
	WholeShared = PackageBox{Count: value}
	pointer.Count++
	return WholeShared.Count, pointer == &WholeShared
}

func Composite(value int32) int32 {
	pointer := &Box{Count: value}
	pointer.Count++
	return pointer.Count
}

func ElidedPointerSlice(value int32) (int32, bool) {
	values := []*Box{{Count: value}, {Count: value + 1}}
	values[0].Count++
	return values[0].Count, values[0] != values[1]
}

func ElidedPointerArray(value int32) int32 {
	values := [1]*Box{{Count: value}}
	values[0].Count++
	return values[0].Count
}

func ElidedPointerMap(value int32) int32 {
	values := map[string]*Box{"value": {Count: value}}
	values["value"].Count++
	return values["value"].Count
}

func ElidedPointerCompositeArray(value int32) int32 {
	values := []*[2]int32{{value, value + 1}}
	values[0][1]++
	return values[0][1]
}

func PointerField(value int32) int32 {
	holder := Holder{Pointer: &Box{Count: value}}
	holder.Pointer.Count++
	return holder.Pointer.Count
}

func PointerToPointer(value int32) int32 {
	pointer := &value
	outer := &pointer
	**outer++
	return value
}

func SliceVariable(value int32) (int32, bool) {
	values := []int32{value}
	pointer := &values
	*pointer = []int32{value + 1}
	return values[0], pointer == &values
}

func MapVariable(value int32) (int32, bool) {
	values := map[int32]int32{1: value}
	pointer := &values
	*pointer = map[int32]int32{1: value + 1}
	return values[1], pointer == &values
}

func plusOne(value int32) int32 {
	return value + 1
}

func plusTwo(value int32) int32 {
	return value + 2
}

func FunctionVariable(value int32) (int32, bool) {
	operation := plusOne
	pointer := &operation
	*pointer = plusTwo
	return operation(value), pointer == &operation
}

func NewMap(value int32) int32 {
	pointer := new(map[int32]int32)
	*pointer = map[int32]int32{1: value}
	return (*pointer)[1]
}

func NewSlice(value int32) int32 {
	pointer := new([]int32)
	*pointer = []int32{value}
	return (*pointer)[0]
}

func NewArray() int32 {
	return (*new([2]int32))[1]
}

func NewStruct() int32 {
	return new(Box).Count
}

func NewPointer() bool {
	return *new(*int32) == nil
}

func ProjectionDoesNotRetarget(value int32) (int32, int32) {
	first := &Box{Count: value}
	second := &Box{Count: value + 10}
	selected := first
	field := &selected.Count
	selected = second
	*field++
	return first.Count, second.Count
}

func AddressOrder(value int32) (int32, int32) {
	var calls int32
	index := func() int {
		calls++
		return 1
	}
	values := [2]int32{value, value + 1}
	pointer := &values[index()]
	*pointer++
	return calls, values[1]
}

func selectBox(box *Box, calls *int32) *Box {
	*calls++
	return box
}

func AddressReceiverOrder(value int32) (int32, int32) {
	var calls int32
	box := &Box{Count: value}
	pointer := &selectBox(box, &calls).Count
	*pointer++
	return calls, box.Count
}

func Cancel(pointer *int32) *int32 {
	return &*pointer
}

func CancelIdentity(value int32) bool {
	pointer := &value
	return Cancel(pointer) == pointer
}

func (box *Box) Add(delta int32) {
	box.Count += delta
}

func (box *Box) Nil() bool {
	return box == nil
}

func (box Box) Incremented() int32 {
	box.Count++
	return box.Count
}

func (box Box) AddressedReceiver() int32 {
	pointer := &box
	pointer.Count++
	return box.Count
}

func PointerReceiverOnValue(value int32) int32 {
	box := Box{Count: value}
	box.Add(3)
	return box.Count
}

func PointerReceiverOnPointer(value int32) int32 {
	box := &Box{Count: value}
	box.Add(4)
	return box.Count
}

func NilPointerReceiver() bool {
	var box *Box
	return box.Nil()
}

func ValueReceiverThroughPointer(value int32) (int32, int32) {
	box := &Box{Count: value}
	result := box.Incremented()
	return result, box.Count
}

func ReceiverStorage(value int32) int32 {
	box := Box{Count: value}
	return box.AddressedReceiver()
}

func ReceiverOrder(value int32) (int32, int32) {
	var calls int32
	box := &Box{Count: value}
	selectBox(box, &calls).Add(2)
	return calls, box.Count
}
