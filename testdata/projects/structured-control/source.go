package structuredcontrol

func Classify(value int) int {
	if current := value; current < 0 {
		return -1
	} else if current == 0 {
		return 0
	} else {
		return 1
	}
}

func Sum(limit int) int {
	total := 0
	current := 0
	for current < limit {
		total = total + current
		current++
	}
	return total
}

func Once() int {
	total := 0
	for {
		total = total + 1
		break
	}
	return total
}
