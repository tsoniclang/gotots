package app

import "example.com/pointer-addressable/dep"

func Mutate(value int32) (int32, bool) {
	pointer := &dep.Shared
	dep.Shared = value
	*pointer++
	return dep.Shared, pointer == &dep.Shared
}
