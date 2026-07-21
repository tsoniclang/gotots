// Package compiler sequences the compilation phases. It holds no semantic
// cases: it wires phase owners and surfaces their typed results. cmd/gotots is
// its only caller, and there is exactly one compilation route.
package compiler

import "github.com/tsoniclang/gotots/internal/language/analyze"

// InspectConstructs runs the construct-inventory phase over one Go file and
// returns the catalog constructs it contains. It fails closed: any construct
// the catalog does not recognize aborts with the classification error.
func InspectConstructs(path string) (analyze.Inventory, error) {
	return analyze.InspectConstructs(path)
}
