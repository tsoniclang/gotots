package deferredfree

func store[T any](target *T, value T) {
	*target = value
}

func direct() (result int32) {
	defer store(&result, int32(42))
	return 1
}

func selected() (result int32) {
	call := store[int32]
	defer call(&result, int32(43))
	return 1
}

func selectStore[T any]() func(*T, T) {
	return store[T]
}

func selectedOpen[T any](value T) (result T) {
	call := selectStore[T]()
	defer call(&result, value)
	return
}

type writer[T any] struct{}

func (writer[T]) store(target *T, value T) {
	*target = value
}

func selectedMethodValue[T any](value T) (result T) {
	var owner writer[T]
	call := owner.store
	defer call(&result, value)
	return
}

func selectedMethodExpression[T any](value T) (result T) {
	var owner writer[T]
	call := writer[T].store
	defer call(owner, &result, value)
	return
}

func selectedLiteral[T any](value T) (result T) {
	call := func(target *T, selected T) {
		*target = selected
	}
	defer call(&result, value)
	return
}

func Audit() int32 {
	return direct() +
		selected() +
		selectedOpen(int32(44)) +
		selectedMethodValue(int32(45)) +
		selectedMethodExpression(int32(46)) +
		selectedLiteral(int32(47))
}
