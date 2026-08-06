package directseams

func Touch() {
}

func Classify(value int32) int32 {
	const low int32 = 3
	const high int32 = 7
	switch {
	case value < low:
		return -1
	case value > high:
		return 1
	default:
		return 0
	}
}

func ClosureCondition(flag bool) int32 {
	changed := false
	set := func() {
		changed = flag
	}
	set()
	switch {
	case !flag:
		return 1
	case changed:
		return 2
	default:
		return 3
	}
}

func Constants() int32 {
	const left, right int32 = 2, 5
	const (
		low  int32 = 3
		high int32 = 7
	)
	return left*1000 + right*100 + low*10 + high
}

func Scoped(value int32) int32 {
	result := value
	if result = result + 1; result > 0 {
		result++
	}
	switch result = result + 1; {
	case result > 5:
		result = result + 10
	default:
		result = result + 20
	}
	return result
}

func ScopedParallel(value int32) int32 {
	left := value
	right := value + 1
	if left, right = right, left; left < right {
		return -1
	}
	switch left, right = right, left; {
	case left < right:
		return left*10 + right
	default:
		return 0
	}
}

func Loop(limit int32) int32 {
	var total int32 = 0
	var current int32 = 0
	for current = 0; current < limit; current = current + 1 {
		total += current
	}
	return total
}

func CallClauses(value int32) int32 {
	if Touch(); value < 0 {
		return value
	}
	for Touch(); value < 2; Touch() {
		value++
	}
	return value
}

func IncInitializers(value int32) int32 {
	if value++; value < 0 {
		return value
	}
	for value++; value < 3; value++ {
	}
	return value
}

func pair() (int32, int32) {
	return 3, 5
}

func add(left, right int32) int32 {
	return left + right
}

func ParallelInitializer() int32 {
	left := int32(1)
	right := int32(2)
	for left, right = right, left; left < 2; left++ {
	}
	return right
}

func ParallelPost(limit int32) int32 {
	left := int32(0)
	right := int32(1)
	total := int32(0)
outer:
	for ; left < limit; left, right = left+1, right+2 {
		if left == 1 {
			continue outer
		}
		total += right
		if left == 3 {
			break outer
		}
	}
	return total*100 + left*10 + right
}

func ConditionPrerequisite(limit int32) int32 {
	count := int32(0)
	for add(pair()) > count {
		count++
		if count == limit {
			break
		}
	}
	return count
}
