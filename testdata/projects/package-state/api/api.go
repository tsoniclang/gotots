package api

import "example.com/package-state/dep"

var Start int32 = dep.Snapshot()

func Run() int32 {
	dep.B++
	dep.Trace++
	Start++
	return Start*10000 + dep.A*1000 + dep.B*100 + dep.Trace
}
