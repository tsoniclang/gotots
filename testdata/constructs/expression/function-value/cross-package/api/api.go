package api

import "example.com/callbackdemand/worker"

func Apply(transform func(int32) int32, value int32) int32 {
	return transform(value)
}

func Run(value int32) int32 {
	return Apply(worker.Double, value)
}
