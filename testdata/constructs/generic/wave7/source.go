package wave7generics

import "example.com/wave7generics/support"

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
	zero := Zero[int32]()
	if !Equal(copied.Get(), int32(9)) ||
		!Equal(alias.Get(), int32(9)) ||
		!Equal(empty.Get(), int32(0)) ||
		!EqualBox(box, copied) ||
		!firstComparable.Same(secondComparable) ||
		!boundSame(secondComparable) ||
		!unboundSame(firstComparable, secondComparable) ||
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
	}
}
