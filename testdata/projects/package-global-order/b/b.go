package b

import (
	"example.com/package-global-order/registry"
	_ "example.com/package-global-order/y"
)

var _ = registry.Mark(2)
