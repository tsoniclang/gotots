package api

import "example.com/control/worker"

func Run(value int) int {
	switch value {
	case 0:
		return 0
	default:
		return worker.Sum(value)
	}
}
