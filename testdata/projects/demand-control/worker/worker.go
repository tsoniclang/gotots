package worker

func Sum(limit int) int {
	total := 0
	for current := 0; current < limit; current++ {
		if current == 2 {
			continue
		}
		total += current
	}
	return total
}

func Unused(value int) int {
	return value + 100
}
