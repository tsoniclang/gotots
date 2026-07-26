package worker

func Sum(limit int32) int32 {
	var total int32 = 0
	for current := total; current < limit; current++ {
		if current == 2 {
			continue
		}
		total += current
	}
	return total
}

func Unused(value int32) int32 {
	return value + 100
}
