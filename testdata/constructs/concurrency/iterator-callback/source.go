package iteratorcallback

func Numbers(limit int32) func(func(int32) bool) {
	return func(yield func(int32) bool) {
		for value := int32(0); value < limit; value++ {
			if !yield(value) {
				return
			}
		}
	}
}

func Words() func(func(string) bool) {
	return func(yield func(string) bool) {
		if !yield("a") {
			return
		}
		yield("bb")
	}
}

func CooperativeAudit() int32 {
	closed := make(chan int32)
	close(closed)
	var total int32
	for value := range Numbers(4) {
		received, ok := <-closed
		if ok {
			panic("closed channel reported a value")
		}
		total += value + received
	}
	return total
}

func SynchronousAudit() int32 {
	var total int32
	for value := range Words() {
		total += int32(len(value))
	}
	return total
}
