// Independent expectation derivation for the source-universe join.
package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// deriveExpectation independently classifies one go list record. std/cmd set
// membership is authoritative; Standard/Goroot corroborate; spelling never
// classifies.
func deriveExpectation(
	pkg *goListPackage,
	stdSet, cmdSet map[string]bool,
	goroot, workspaceDir string,
	overlay map[string][]byte,
) (*universeExpectation, identity.PackageID, error) {
	out := &universeExpectation{
		root: !pkg.DepOnly, disposition: source.DispositionOrdinarySource,
		name:      pkg.Name,
		files:     map[identity.FileID]bool{},
		filePaths: map[identity.FileID]string{},
		inputs:    map[string]bool{}, embedPatterns: map[string]bool{},
		importPaths: map[string]bool{},
		imports:     map[identity.PackageID]bool{},
		cgoSources:  map[identity.FileID]bool{},
	}
	for _, imported := range pkg.Imports {
		if imported != "C" {
			out.importPaths[imported] = true
		}
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
			return nil, identity.PackageID{}, fmt.Errorf("%s: std-set member without corroborating Standard/Goroot facts", pkg.ImportPath)
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
			return nil, identity.PackageID{}, err
		}
		owner, err = identity.NewModuleOwner(moduleID)
		if err != nil {
			return nil, identity.PackageID{}, err
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
		return nil, identity.PackageID{}, fmt.Errorf("%s: toolchain names a package that is neither module-owned nor std/cmd", pkg.ImportPath)
	}
	packageID, err := identity.NewPackageID(owner, pkg.ImportPath)
	if err != nil {
		return nil, identity.PackageID{}, err
	}
	if out.disposition == source.DispositionOrdinarySource {
		out.initialization = source.PackageInitializationGo
	} else {
		out.initialization = source.PackageInitializationNone
	}
	for _, goFile := range append(append([]string{}, pkg.GoFiles...), pkg.CgoFiles...) {
		absolute := absOrJoin(pkg.Dir, goFile)
		rel, err := filepath.Rel(relBase, absolute)
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, identity.PackageID{}, fmt.Errorf("%s: file %s outside owner root", pkg.ImportPath, goFile)
		}
		fileID, err := identity.NewFileID(owner, filepath.ToSlash(rel))
		if err != nil {
			return nil, identity.PackageID{}, err
		}
		out.files[fileID] = true
		out.filePaths[fileID] = absolute
	}
	sourcePaths := map[string]bool{}
	for _, path := range append(
		append([]string{}, pkg.GoFiles...), pkg.CgoFiles...,
	) {
		sourcePaths[filepath.Clean(absOrJoin(pkg.Dir, path))] = true
	}
	compiledPaths := map[string]bool{}
	for _, path := range pkg.CompiledGoFiles {
		compiledPaths[filepath.Clean(absOrJoin(pkg.Dir, path))] = true
	}
	if out.disposition == source.DispositionOrdinarySource {
		for path := range sourcePaths {
			out.checkedView = out.checkedView || !compiledPaths[path]
		}
		for path := range compiledPaths {
			out.checkedView = out.checkedView || !sourcePaths[path]
		}
	}
	for _, cgoFile := range pkg.CgoFiles {
		rel, err := filepath.Rel(relBase, absOrJoin(pkg.Dir, cgoFile))
		if err == nil && !strings.HasPrefix(rel, "..") {
			if fileID, err := identity.NewFileID(owner, filepath.ToSlash(rel)); err == nil {
				out.cgoSources[fileID] = true
			}
		}
	}
	inputGroups := []struct {
		kind  source.InputKind
		files []string
	}{
		{source.InputC, pkg.CFiles},
		{source.InputCXX, pkg.CXXFiles},
		{source.InputObjectiveC, pkg.MFiles},
		{source.InputHeader, pkg.HFiles},
		{source.InputFortran, pkg.FFiles},
		{source.InputAssembly, pkg.SFiles},
		{source.InputSWIG, pkg.SwigFiles},
		{source.InputSWIGCXX, pkg.SwigCXXFiles},
		{source.InputSyso, pkg.SysoFiles},
	}
	for _, group := range inputGroups {
		for _, input := range group.files {
			if err := addExpectedInput(
				out,
				owner,
				relBase,
				pkg.Dir,
				input,
				group.kind,
				overlay,
			); err != nil {
				return nil, identity.PackageID{}, err
			}
		}
	}
	for _, embed := range pkg.EmbedFiles {
		if err := addExpectedInput(
			out,
			owner,
			relBase,
			pkg.Dir,
			embed,
			source.InputEmbed,
			overlay,
		); err != nil {
			return nil, identity.PackageID{}, err
		}
	}
	for _, pattern := range pkg.EmbedPatterns {
		out.embedPatterns[pattern] = true
	}
	return out, packageID, nil
}

func addExpectedInput(
	out *universeExpectation,
	owner identity.Owner,
	relBase string,
	packageDir string,
	path string,
	kind source.InputKind,
	overlay map[string][]byte,
) error {
	path = absOrJoin(packageDir, path)
	rel, err := filepath.Rel(relBase, path)
	if err != nil || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("supplemental input %s lies outside owner root", path)
	}
	id, err := identity.NewFileID(owner, filepath.ToSlash(rel))
	if err != nil {
		return err
	}
	raw, overlaid := overlay[path]
	if !overlaid {
		raw, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf(
				"supplemental input %s unreadable: %w", id, err,
			)
		}
	}
	key := fmt.Sprintf(
		"%s|%s|%x|%t", kind, id, sha256.Sum256(raw), overlaid,
	)
	if out.inputs[key] {
		return fmt.Errorf("duplicate supplemental input %s", id)
	}
	out.inputs[key] = true
	return nil
}

func absOrJoin(dir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(dir, path)
}
