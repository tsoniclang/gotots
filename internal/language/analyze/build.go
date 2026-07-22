package analyze

import (
	"github.com/tsoniclang/gotots/internal/source"
)

// BuildFileInventory produces the immutable inventory artifact of one loaded
// file: catalog-driven traversal, variant resolution, implicit-operation
// detection, and directive inventory, then constructor validation. Any failure
// aborts the artifact (fail closed).
func BuildFileInventory(pkg *source.Package, file *source.File) (*FileInventory, error) {
	b := &builder{fset: file.Fset(), file: file.ID()}
	if err := b.visit(file.Syntax(), -1, 0); err != nil {
		return nil, err
	}
	if err := resolveVariants(b, pkg.TypesInfo()); err != nil {
		return nil, err
	}
	detectImplicit(b, pkg.TypesInfo())
	directives, err := scanDirectives(b, file)
	if err != nil {
		return nil, err
	}
	return newFileInventory(file.Path(), file.ID(), b.occurrences, directives)
}

// BuildWorkspaceInventory produces the immutable whole-workspace inventory
// artifact over a loaded source universe.
func BuildWorkspaceInventory(ws *source.Workspace) (*WorkspaceInventory, error) {
	out := &WorkspaceInventory{version: InventoryArtifactVersion, goVersion: ws.GoVersion()}
	for _, pkg := range ws.Packages() {
		pkgInventory := &PackageInventory{id: pkg.ID()}
		for _, file := range pkg.Files() {
			fileInventory, err := BuildFileInventory(pkg, file)
			if err != nil {
				return nil, err
			}
			pkgInventory.files = append(pkgInventory.files, fileInventory)
		}
		out.packages = append(out.packages, pkgInventory)
	}
	return out, nil
}
