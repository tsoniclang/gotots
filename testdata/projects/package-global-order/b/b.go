package b

import (
	_ "example.com/package-global-order/y"
	"example.com/package-global-order/registry"
)

var _ = registry.Mark(2)
