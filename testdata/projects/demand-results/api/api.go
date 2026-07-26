package api

import "example.com/results/producer"

func Run(value int) int {
	next, zero := producer.Pair(value)
	if zero {
		return next
	}
	return value
}
