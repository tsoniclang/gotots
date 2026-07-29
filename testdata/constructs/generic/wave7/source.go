package wave7generics

type Integer interface {
	~int32 | ~int64
}

type ShiftCount interface {
	~uint8 | ~uint16
}

type Box[T any] struct {
	Value T
}

type Alias[T any] = Box[T]

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
	copied := CopyBox(box)
	empty := ZeroBox[int32]()
	zero := Zero[int32]()
	if !Equal(copied.Get(), int32(9)) ||
		!Equal(empty.Get(), int32(0)) ||
		!EqualBox(box, copied) {
		return []int32{-1}
	}
	return []int32{box.Get(), zero}
}
