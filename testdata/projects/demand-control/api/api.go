package api

import "example.com/control/worker"

func Run(value int32) int32 {
	switch value {
	case 0:
		return 0
	default:
		return worker.Sum(value)
	}
}
