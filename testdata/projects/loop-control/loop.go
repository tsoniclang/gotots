package loopcontrol

func Sum(limit int32) int32 {
	var total int32 = 0
	for current := total; current < limit; current++ {
		if current == 2 {
			continue
		}
		total = total + current
		if total > 10 {
			break
		}
	}
	return total
}
