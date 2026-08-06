package crosslocal

import "example.com/crosslocal/api"

func Audit() int32 {
	type Local int32
	value := api.Twice(Local(21))
	type Outer int32
	var nested int32
	{
		type Inner int32
		first := api.BothEqual(Outer(20), Outer(20), Inner(1), Inner(1))
		second := api.BothEqual(Outer(1), Outer(1), Inner(2), Inner(2))
		if first && second {
			nested = 21
		}
	}
	return int32(value) + nested
}
