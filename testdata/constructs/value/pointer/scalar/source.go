package scalarpointer

var shared *int32

func NewValue(value int32) *int32 {
	pointer := new(int32)
	*pointer = value
	return pointer
}

func NewZero() int32 {
	return *new(int32)
}

func Bool(value bool) bool {
	pointer := new(bool)
	*pointer = value
	return *pointer
}

func Wide(value int64) int64 {
	pointer := new(int64)
	*pointer = value
	return *pointer
}

func Read(pointer *int32) int32 {
	return *pointer
}

func Store(pointer *int32, value int32) {
	*pointer = value
}

func Forward(pointer *int32) *int32 {
	return pointer
}

func Alias(value int32) (int32, bool, bool) {
	original := new(int32)
	alias := original
	var assigned *int32
	assigned = alias
	Store(assigned, value)
	forwarded := Forward(original)
	return Read(forwarded), original == alias, original != nil
}

func Zero() *int32 {
	var pointer *int32
	return pointer
}

func IsNil(pointer *int32) bool {
	return nil == pointer
}

func Reset(pointer *int32) bool {
	pointer = nil
	return pointer == nil
}

func Distinct() bool {
	return new(int32) != new(int32)
}

func SetShared(pointer *int32) {
	shared = pointer
}

func SharedIsNil() bool {
	return shared == nil
}

func SharedValue() int32 {
	return *shared
}

func NilRead() int32 {
	var pointer *int32
	return *pointer
}

func Ordinary(value int32) int32 {
	local := value
	copy := local
	return local + copy
}
