package cooperativeforclause

type Predicate interface {
	More() bool
}

type counterPredicate struct {
	closed    chan int32
	remaining []int32
}

func (predicate counterPredicate) More() bool {
	<-predicate.closed
	if predicate.remaining[0] == 0 {
		return false
	}
	predicate.remaining[0]--
	return true
}

func selectPredicate(predicate Predicate) Predicate {
	return predicate
}

func next(closed chan int32, value int32) (int32, bool) {
	<-closed
	return value, value < 3
}

func CooperativeCondition() int32 {
	closed := make(chan int32)
	close(closed)
	predicate := counterPredicate{
		closed:    closed,
		remaining: []int32{3},
	}
	var count int32
	for selectPredicate(predicate).More() {
		count++
	}
	return count
}

func CooperativePost() int32 {
	closed := make(chan int32)
	close(closed)
	value, ok := next(closed, 0)
	var total int32
	for ; ok; value, ok = next(closed, value+1) {
		if value == 1 {
			continue
		}
		total += value
	}
	return total
}
