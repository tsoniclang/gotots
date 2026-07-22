package analyze

import (
	"github.com/tsoniclang/gotots/internal/source"
)

// BuildFileInventory produces the immutable inventory artifact of one loaded
// file: catalog-driven traversal, variant resolution, implicit-operation
// detection, and directive inventory, then constructor validation. Any failure
// aborts the artifact (fail closed).
func BuildFileInventory(pkg *source.Package, file *source.File) (*FileInventory, error) {
	syntax, ok := file.FullSyntax()
	if !ok {
		return nil, newResolutionError(0, file.ID(), Span{}, "inventory requires full-syntax evidence")
	}
	b := &builder{fset: file.Fset(), file: file.ID()}
	if err := b.visit(syntax, -1, 0); err != nil {
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
	return newFileInventory(file.Path(), file.ID(), file.EffectiveGoVersion(), b.occurrences, directives)
}

// BuildWorkspaceInventory produces the immutable inventory artifact over
// exactly the full-semantic scope of the finalized universe: whole files
// whose every unit is full-semantic, and the retained unit subtrees of mixed
// files. Declaration-contract, external-boundary, and intrinsic evidence
// contributes boundary records in the source artifact, never interior
// occurrences; the retention decision is the scope phase's, never made here.
func BuildWorkspaceInventory(ws *source.Workspace) (*WorkspaceInventory, error) {
	out := &WorkspaceInventory{version: InventoryArtifactVersion}
	for _, pkg := range ws.Packages() {
		if !pkg.RetainsFullSemantic() {
			continue
		}
		pkgInventory := &PackageInventory{id: pkg.ID()}
		for _, file := range pkg.Files() {
			switch evidence := file.Evidence().(type) {
			case source.FullSyntax:
				fileInventory, err := BuildFileInventory(pkg, file)
				if err != nil {
					return nil, err
				}
				pkgInventory.files = append(pkgInventory.files, fileInventory)
			case source.MixedUnits:
				for _, retained := range evidence.Retained {
					unitInventory, err := buildUnitInventory(pkg, file, retained)
					if err != nil {
						return nil, err
					}
					pkgInventory.files = append(pkgInventory.files, unitInventory)
				}
			case source.ContractOnly:
				// boundary-only evidence; no interior occurrences
			}
		}
		if len(pkgInventory.files) > 0 {
			out.packages = append(out.packages, pkgInventory)
		}
	}
	return out, nil
}

// buildUnitInventory inventories one retained full-semantic unit subtree of a
// mixed file, rooted at its declaration.
func buildUnitInventory(pkg *source.Package, file *source.File, retained source.RetainedUnit) (*FileInventory, error) {
	b := &builder{fset: file.Fset(), file: file.ID()}
	if err := b.visit(retained.Decl, -1, 0); err != nil {
		return nil, err
	}
	if err := resolveVariants(b, pkg.TypesInfo()); err != nil {
		return nil, err
	}
	detectImplicit(b, pkg.TypesInfo())
	rootSpan := Span{
		Start: Position{Offset: retained.Unit.Span().Start()},
		End:   Position{Offset: retained.Unit.Span().End()},
	}
	if retained.FromCheckedView {
		rootSpan = Span{
			Start: Position{Offset: retained.CheckedSpan.Start.Offset},
			End:   Position{Offset: retained.CheckedSpan.End.Offset},
		}
	}
	return newUnitInventory(file.Path(), file.ID(), file.EffectiveGoVersion(), retained.Unit, rootSpan, b.occurrences)
}
