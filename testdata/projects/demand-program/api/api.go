package api

import "example.com/demand/service"

const Compute int32 = 5

func Run(value int32) int32 {
	return service.Compute(value) + Compute
}

func unusedAPI(value int32) int32 {
	return value + 100
}
