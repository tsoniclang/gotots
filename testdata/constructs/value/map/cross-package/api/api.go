package api

import "example.com/mapcross/store"

func Run(value int32) (int32, bool) {
	values := store.New(value)
	alias := store.Identity(values)
	alias[2] = value + 1
	first, ok := values[1]
	return first + values[2], ok
}
