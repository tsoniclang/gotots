package service

import "example.com/demand/mathx"

func Compute(value int) int {
	Even := value
	Even += mathx.Even(value)
	return Even
}

func UnusedService(value int) int {
	return value + 200
}
