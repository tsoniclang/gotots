package support

type Holder[T any] struct {
	Value T
}

func Make[T any](value T) Holder[T] {
	return Holder[T]{Value: value}
}

func (holder Holder[T]) Get() T {
	return holder.Value
}
