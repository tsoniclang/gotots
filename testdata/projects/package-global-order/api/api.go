package api

import (
	_ "example.com/package-global-order/a"
	_ "example.com/package-global-order/b"
	"example.com/package-global-order/registry"
)

var Observed int32 = registry.Read()

func Run() int32 {
	return Observed
}
