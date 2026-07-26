package a

import (
	"example.com/package-global-order/registry"
	_ "example.com/package-global-order/z"
)

var _ = registry.Mark(4)
