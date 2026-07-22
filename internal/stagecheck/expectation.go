// Independent expectation derivation for the source-universe join.
package stagecheck

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// deriveExpectation independently classifies one go list record. std/cmd set
// membership is authoritative; Standard/Goroot corroborate; spelling never
// classifies.
func deriveExpectation(pkg *goListPackage, stdSet, cmdSet map[string]bool, goroot, workspaceDir string) (*universeExpectation, string, error) {
	out := &universeExpectation{
		root: !pkg.DepOnly, disposition: source.DispositionOrdinarySource,
		files: map[string]bool{}, otherFiles: map[string]bool{},
		embedFiles: map[string]bool{}, embedPatterns: map[string]bool{},
		cgoSources: map[string]bool{},
	}
	if pkg.Module != nil {
		out.moduleGo = pkg.Module.GoVersion
	}
	var owner identity.Owner
	var relBase string
	switch {
	case pkg.ImportPath == "builtin":
		owner = identity.LanguagePseudoOwner()
		out.provenance, out.acquisition, out.disposition = source.ProvenanceLanguagePseudo, source.AcquisitionGOROOT, source.DispositionBuiltinUniverse
		out.moduleGo = ""
		relBase = filepath.Join(goroot, "src")
	case stdSet[pkg.ImportPath]:
		if !pkg.Standard || !pkg.Goroot {
			return nil, "", fmt.Errorf("%s: std-set member without corroborating Standard/Goroot facts", pkg.ImportPath)
		}
		owner = identity.StandardLibraryOwner()
		out.provenance, out.acquisition = source.ProvenanceStandardLibrary, source.AcquisitionGOROOT
		if pkg.ImportPath == "unsafe" {
			out.disposition = source.DispositionUnsafeIntrinsic
		}
		out.moduleGo = ""
		relBase = filepath.Join(goroot, "src")
	case cmdSet[pkg.ImportPath]:
		// cmd-set membership is authoritative; there is no GOROOT fallback.
		owner = identity.ToolchainOwner()
		out.provenance, out.acquisition = source.ProvenanceToolchainPackage, source.AcquisitionGOROOT
		out.moduleGo = ""
		relBase = filepath.Join(goroot, "src")
	case pkg.Module != nil:
		moduleDir := pkg.Module.Dir
		if pkg.Module.Replace != nil && pkg.Module.Replace.Dir != "" {
			moduleDir = pkg.Module.Replace.Dir
		}
		version := pkg.Module.Version
		if pkg.Module.Main {
			version = ""
		}
		moduleID, err := identity.NewModuleID(pkg.Module.Path, version)
		if err != nil {
			return nil, "", err
		}
		owner, err = identity.NewModuleOwner(moduleID)
		if err != nil {
			return nil, "", err
		}
		switch {
		case pkg.Module.Main:
			out.provenance, out.acquisition = source.ProvenanceWorkspaceModule, source.AcquisitionWorkspace
		case pkg.Module.Replace != nil && pkg.Module.Replace.Version == "":
			out.provenance, out.acquisition = source.ProvenanceModuleDependency, source.AcquisitionLocalReplacement
		case moduleDir == "":
			out.provenance, out.acquisition = source.ProvenanceModuleDependency, source.AcquisitionVendor
		default:
			out.provenance = source.ProvenanceModuleDependency
			if under(pkg.Dir, workspaceDir) && strings.Contains(filepath.ToSlash(pkg.Dir), "/vendor/") {
				out.acquisition = source.AcquisitionVendor
			} else {
				out.acquisition = source.AcquisitionModuleCache
			}
		}
		relBase = moduleDir
		if relBase == "" {
			marker := filepath.FromSlash("vendor/" + pkg.Module.Path)
			if idx := strings.Index(pkg.Dir, marker); idx >= 0 {
				relBase = pkg.Dir[:idx+len(marker)]
			} else {
				relBase = workspaceDir
			}
		}
	default:
		return nil, "", fmt.Errorf("%s: toolchain names a package that is neither module-owned nor std/cmd", pkg.ImportPath)
	}
	packageID, err := identity.NewPackageID(owner, pkg.ImportPath)
	if err != nil {
		return nil, "", err
	}
	for _, goFile := range append(append([]string{}, pkg.GoFiles...), pkg.CgoFiles...) {
		rel, err := filepath.Rel(relBase, filepath.Join(pkg.Dir, goFile))
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, "", fmt.Errorf("%s: file %s outside owner root", pkg.ImportPath, goFile)
		}
		fileID, err := identity.NewFileID(owner, filepath.ToSlash(rel))
		if err != nil {
			return nil, "", err
		}
		out.files[fileID.String()] = true
	}
	for _, cgoFile := range pkg.CgoFiles {
		rel, err := filepath.Rel(relBase, filepath.Join(pkg.Dir, cgoFile))
		if err == nil && !strings.HasPrefix(rel, "..") {
			if fileID, err := identity.NewFileID(owner, filepath.ToSlash(rel)); err == nil {
				out.cgoSources[fileID.String()] = true
			}
		}
	}
	for _, other := range pkg.otherFiles() {
		out.otherFiles[ownerRelOrRaw(relBase, filepath.Join(pkg.Dir, other))] = true
	}
	for _, embed := range pkg.EmbedFiles {
		out.embedFiles[ownerRelOrRaw(relBase, absOrJoin(pkg.Dir, embed))] = true
	}
	for _, pattern := range pkg.EmbedPatterns {
		out.embedPatterns[pattern] = true
	}
	return out, packageID.String(), nil
}

func ownerRelOrRaw(relBase, path string) string {
	if rel, err := filepath.Rel(relBase, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}

func absOrJoin(dir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}
