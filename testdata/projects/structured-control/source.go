package structuredcontrol

func Classify(value int32) int32 {
	if current := value; current < 0 {
		return -1
	} else if current == 0 {
		return 0
	} else {
		return 1
	}
}

func Sum(limit int32) int32 {
	var total int32 = 0
	var current int32 = 0
	for current < limit {
		total = total + current
		current++
	}
	return total
}

func Once() int32 {
	var total int32 = 0
	for {
		total = total + 1
		break
	}
	return total
}
