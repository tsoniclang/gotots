package source

import "fmt"

// CheckTypeGraphCoherence proves the transient universe is one coherent
// go/types graph: every import edge resolves to the identical *types.Package
// object stored on the imported record, never mixed loader calls. It runs on
// the transient graph (before finalization severs it); the finalized artifact
// exposes no raw *types.Package.
func (u *Universe) CheckTypeGraphCoherence() error {
	byPath := map[string]*LoadedPackage{}
	for _, pkg := range u.packages {
		byPath[pkg.id.ImportPath()] = pkg
	}
	for _, pkg := range u.packages {
		if pkg.types == nil {
			continue
		}
		for _, imported := range pkg.types.Imports() {
			record, tracked := byPath[imported.Path()]
			if !tracked || record.types == nil {
				continue
			}
			if record.types != imported {
				return &LoadError{Dir: u.request.Dir, Reason: fmt.Sprintf(
					"type-graph incoherence: %s imports %s as a distinct types.Package object",
					pkg.id, imported.Path())}
			}
		}
	}
	return nil
}
