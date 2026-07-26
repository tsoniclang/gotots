package api

import "example.com/results/producer"

func Run(value int32) int32 {
	next, zero := producer.Pair(value)
	if zero {
		return next
	}
	return value
}
