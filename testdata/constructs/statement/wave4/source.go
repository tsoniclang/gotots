package wave4statements

type Count int32

type Box struct {
	Value int32
}

func pair() (int32, int32) {
	return 3, 5
}

func arrayAndSlice() []int32 {
	array := [3]int32{1, 2, 3}
	result := make([]int32, 0, 16)
	for index, value := range array {
		if index == 0 {
			array[1] = 90
		}
		result = append(result, int32(index), value)
	}

	slice := []int32{4, 5, 6}
	for index, value := range slice {
		if index == 0 {
			slice[1] = 50
		}
		result = append(result, int32(index), value)
	}

	pointer := &[3]int32{7, 8, 9}
	for index, value := range pointer {
		if index == 0 {
			pointer[1] = 80
		}
		result = append(result, int32(index), value)
	}
	return result
}

func constantLengthDoesNotEvaluate() int32 {
	var pointer *[4]int32
	var sum int32
	for index := range pointer {
		sum += int32(index)
	}
	return sum
}

func nonconstantArraySourceEvaluates() int32 {
	var calls int32
	makeArray := func() [2]int32 {
		calls++
		return [2]int32{3, 4}
	}
	var sum int32
	for index := range makeArray() {
		sum += int32(index)
	}
	return calls*10 + sum
}

func perIterationVariables() int32 {
	var pointers []*int32
	for _, value := range []int32{10, 20, 30} {
		pointers = append(pointers, &value)
	}
	return *pointers[0] + *pointers[1] + *pointers[2]
}

func assignmentRange() int32 {
	values := []int32{2, 4, 8}
	var index int
	var value int32
	var sum int32
	for index, value = range values {
		sum += int32(index) + value
	}
	return sum
}

func stringRange() []int32 {
	value := string([]byte{
		'a',
		0xc3, 0xa9,
		0xff,
		0xf0, 0x9f, 0x99, 0x82,
	})
	result := make([]int32, 0, 10)
	for index, runeValue := range value {
		result = append(result, int32(index), runeValue)
	}
	return result
}

func mapRange() int32 {
	values := map[[2]int32]Box{
		{1, 2}: {Value: 3},
		{4, 5}: {Value: 6},
	}
	var count int32
	var sum int32
	for key, value := range values {
		count++
		sum += key[0] + key[1] + value.Value
	}
	return count*100 + sum
}

func mapDeletionRange() int32 {
	values := map[int32]int32{1: 10, 2: 20, 3: 30}
	var count int32
	for key := range values {
		count++
		for candidate := range values {
			if candidate != key {
				delete(values, candidate)
			}
		}
	}
	return count
}

func assignmentTargetRange() int32 {
	values := []int32{2, 4}
	targets := []int{0, 0}
	var calls int32
	next := func() int {
		index := calls
		calls++
		return int(index)
	}
	var value int32
	for targets[next()], value = range values {
	}
	return calls*100 + int32(targets[1])*10 + value
}

func integerRange(limit Count) int32 {
	var sum int32
outer:
	for index := range limit {
		if index == 2 {
			continue outer
		}
		sum += int32(index)
		if index == 4 {
			break outer
		}
	}
	return sum
}

func noVariableRange() int32 {
	var count int32
	for range [3]int32{} {
		count++
	}
	return count
}

func switchFallthrough(value int32) int32 {
	switch value {
	case 1:
		value += 10
		fallthrough
	case 2, 3:
		value += 20
	default:
		value += 30
	}
	return value
}

func switchDefaultFallthrough(value int32) int32 {
	switch value {
	default:
		value += 10
		fallthrough
	case 1:
		value += 20
	}
	return value
}

func switchArray(value [2]int32) int32 {
	switch value {
	case [2]int32{1, 2}:
		return 12
	case [2]int32{3, 4}:
		value[0] += 10
		fallthrough
	default:
		return value[0] + value[1]
	}
	return 0
}

func add(left, right int32) int32 {
	return left + right
}

func switchPrerequisite(value int32) int32 {
	switch value {
	case add(pair()):
		return 1
	default:
		return 0
	}
	return 0
}

func localForms() int32 {
	var left, right = pair()
	var (
		zero int32
		_    = right
	)

	return left + right + zero
}

func Audit() []int32 {
	result := arrayAndSlice()
	result = append(
		result,
		constantLengthDoesNotEvaluate(),
		nonconstantArraySourceEvaluates(),
		perIterationVariables(),
		assignmentRange(),
		mapRange(),
		mapDeletionRange(),
		assignmentTargetRange(),
		integerRange(7),
		integerRange(-3),
		noVariableRange(),
		switchFallthrough(1),
		switchFallthrough(2),
		switchFallthrough(9),
		switchDefaultFallthrough(1),
		switchDefaultFallthrough(2),
		switchArray([2]int32{1, 2}),
		switchArray([2]int32{3, 4}),
		switchPrerequisite(8),
		switchPrerequisite(9),
		localForms(),
	)
	result = append(result, stringRange()...)
	return result
}
