package api

import "example.com/constantcross/defs"

func Widths() (int8, int32, uint16) {
	return defs.Width, defs.Width, defs.Width
}

func Enum() (int, int, int) {
	return defs.First, defs.Second, defs.Third
}

func Flags() (bool, string) {
	return defs.Enabled, defs.Label
}
