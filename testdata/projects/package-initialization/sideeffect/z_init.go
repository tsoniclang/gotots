package sideeffect

import "example.com/package-initialization/sink"

func init() {
	sink.Mark(7)
}
