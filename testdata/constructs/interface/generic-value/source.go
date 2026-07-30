package genericinterface

type Value[T any] interface {
	Get() T
}

type Cloneable[T any] interface {
	Clone() T
}

type Box struct {
	Value int32
}

func (box Box) Get() int32 {
	return box.Value
}

type StringBox struct{}

func (box StringBox) Get() string {
	return "wrong"
}

type Mutable struct {
	Value int32
}

func (value *Mutable) Clone() *Mutable {
	return &Mutable{Value: value.Value + 1}
}

func Clone[T Cloneable[T]](value T) T {
	return value.Clone()
}

func Read(value Value[int32]) int32 {
	return value.Get()
}

func IsNil(value Value[int32]) bool {
	return value == nil
}

func Assert(value any) (Value[int32], bool) {
	result, ok := value.(Value[int32])
	return result, ok
}

func Audit() int32 {
	var value Value[int32] = Box{Value: 41}
	if IsNil(value) || !IsNil(nil) {
		return -1
	}
	accepted, ok := Assert(value)
	if !ok {
		return -2
	}
	if _, ok := Assert(StringBox{}); ok {
		return -3
	}
	cloned := Clone(&Mutable{Value: Read(accepted)})
	return cloned.Value
}
