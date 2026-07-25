package loopcontrol

func Sum(limit int) int {
	total := 0
	for current := 0; current < limit; current++ {
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
