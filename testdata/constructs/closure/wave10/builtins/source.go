package builtins

func Collections() int32 {
	values := make([]int32, 2, 4)
	values = append(values, 3)
	target := make([]int32, len(values))
	count := copy(target, values)
	capacity := cap(values)
	clear(values)

	mapping := make(map[int32]int32)
	mapping[1] = 2
	delete(mapping, 1)
	clear(mapping)

	pointer := new(int32)
	*pointer = int32(count + capacity)
	return *pointer
}

func ComplexParts() float64 {
	value := complex(3.0, 4.0)
	return real(value) + imag(value)
}

func Ordered(left, right int32) int32 {
	return min(left, right) + max(left, right)
}

func CloseChannel() {
	values := make(chan int32, 1)
	close(values)
}

func recoverInto(target *any) {
	*target = recover()
}

func PanicAndRecover() (result any) {
	defer recoverInto(&result)
	panic("wave10")
}
