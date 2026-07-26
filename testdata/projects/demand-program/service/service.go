package service

import "example.com/demand/mathx"

func Compute(value int32) int32 {
	Even := value
	Even += mathx.Even(value)
	return Even
}

func UnusedService(value int32) int32 {
	return value + 200
}
