package api

import "example.com/demand/service"

const Compute int = 5

func Run(value int) int {
	return service.Compute(value) + Compute
}

func unusedAPI(value int) int {
	return value + 100
}
