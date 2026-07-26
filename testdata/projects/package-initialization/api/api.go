package api

import (
	_ "example.com/package-initialization/sideeffect"
	"example.com/package-initialization/sink"
)

var Observed int32 = sink.Read()

func Run() int32 {
	return Observed + sink.Read()
}
