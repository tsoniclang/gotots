package sideeffect

import "example.com/package-initialization/sink"

var _ = sink.Mark(3)
var first, second = sink.Pair()
