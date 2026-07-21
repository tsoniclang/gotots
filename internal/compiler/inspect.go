// Package compiler sequences the compilation phases. It holds no semantic
// cases: it orders the phase owners and surfaces their typed results. cmd/gotots
// is its only caller, and there is exactly one compilation route.
package compiler

import (
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/source"
)

// InspectConstructs orchestrates the workspace-load and construct-inventory
// phases over one Go file: source loads and parses the file into a typed
// artifact, then analysis inventories its constructs. It fails closed on either
// a load error or an unrecognized construct.
func InspectConstructs(path string) (analyze.Inventory, error) {
	loaded, err := source.Load(path)
	if err != nil {
		return analyze.Inventory{}, err
	}
	return analyze.BuildInventory(loaded)
}
