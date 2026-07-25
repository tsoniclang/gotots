package source

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

// ForkForHydration creates an isolated transient view of one resolved
// universe. Only the named packages are cloned because selective hydration
// mutates only its initial package records; all other resolution records are
// immutable metadata shared read-only. This is used by disk-backed provider
// audit, one package at a time.
func ForkForHydration(
	universe *Universe,
	packageIDs []identity.PackageID,
) (*Universe, error) {
	if universe == nil || universe.hydrated || universe.finalized {
		return nil, &LoadError{
			Reason: "hydration fork requires an unresolved transient universe",
		}
	}
	selected := map[identity.PackageID]bool{}
	for _, packageID := range packageIDs {
		if packageID.IsZero() || selected[packageID] {
			return nil, &LoadError{
				Reason: "hydration fork has an invalid or duplicate package",
			}
		}
		selected[packageID] = true
	}
	fork := &Universe{
		toolchain: universe.toolchain,
		request:   cloneRequest(universe.request),
	}
	replacements := map[*LoadedPackage]*LoadedPackage{}
	for _, pkg := range universe.packages {
		record := pkg
		if selected[pkg.id] {
			record = cloneLoadedPackageMetadata(pkg)
			replacements[pkg] = record
			delete(selected, pkg.id)
		}
		fork.packages = append(fork.packages, record)
	}
	if len(selected) != 0 {
		for packageID := range selected {
			return nil, &LoadError{
				Reason: fmt.Sprintf(
					"hydration fork names unknown package %s", packageID,
				),
			}
		}
	}
	for _, root := range universe.roots {
		if replacement := replacements[root]; replacement != nil {
			fork.roots = append(fork.roots, replacement)
		} else {
			fork.roots = append(fork.roots, root)
		}
	}
	return fork, nil
}

func cloneLoadedPackageMetadata(
	pkg *LoadedPackage,
) *LoadedPackage {
	out := *pkg
	out.imports = append([]PackageImport(nil), pkg.imports...)
	out.inputs = append([]loadedInput(nil), pkg.inputs...)
	out.embedPatterns = append([]string(nil), pkg.embedPatterns...)
	out.types = nil
	out.typesInfo = nil
	out.checkedDecls = nil
	out.files = make([]*LoadedFile, 0, len(pkg.files))
	for _, file := range pkg.files {
		cloned := *file
		cloned.fset = nil
		cloned.syntax = nil
		cloned.checkerFile = nil
		cloned.physicalFset = nil
		cloned.physicalSyntax = nil
		cloned.selectedBytes = nil
		out.files = append(out.files, &cloned)
	}
	return &out
}

// DiscardHydratedUniverse actively severs a package-audit fork after its
// immutable package shard has been written.
func DiscardHydratedUniverse(universe *Universe) error {
	if universe == nil || !universe.hydrated || universe.finalized {
		return &LoadError{
			Reason: "discard requires one live hydrated universe",
		}
	}
	severTransientGraph(universe)
	return nil
}
