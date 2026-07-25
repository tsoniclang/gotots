package source

import (
	"crypto/sha256"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/identity"
)

// hydrationMode creates the one coherent checker graph required by local
// structural decisions. NeedDeps is deliberately absent: imported packages
// are loaded from export data instead of parsing their interiors.
const hydrationMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedModule |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedEmbedFiles |
	packages.NeedEmbedPatterns

// HydrateUniverse performs the sole semantic load after source planning. The
// request is exact: only packages owning local files or selected checked-view
// syntax are initial checker packages. Imported type nodes come from export
// data and share the same checker graph; their source bodies are not parsed.
func HydrateUniverse(
	universe *Universe,
	request HydrationRequest,
) (resultErr error) {
	if universe == nil {
		return invalidHydration("universe is nil")
	}
	if universe.hydrated {
		return invalidHydration("universe is already hydrated")
	}
	if universe.finalized {
		return invalidHydration("universe is finalized")
	}
	files, synthetic, err := resolveHydrationDemand(universe, request)
	if err != nil {
		return err
	}
	if len(files) == 0 && len(synthetic) == 0 {
		universe.hydrationOwners = map[identity.PackageID]bool{}
		universe.hydrated = true
		return nil
	}
	packageFor := packageForFile(universe)
	initial := map[identity.PackageID]*LoadedPackage{}
	for fileID := range files {
		initial[packageFor[fileID].id] = packageFor[fileID]
	}
	for packageID := range synthetic {
		initial[packageID] = synthetic[packageID]
	}
	universe.hydrationOwners = map[identity.PackageID]bool{}
	for packageID := range initial {
		universe.hydrationOwners[packageID] = true
	}
	defer func() {
		if resultErr != nil {
			clearTransientEvidence(universe)
		}
	}()
	patterns := make([]string, 0, len(initial))
	for _, pkg := range initial {
		patterns = append(patterns, pkg.id.ImportPath())
	}
	sort.Strings(patterns)
	toolchain, env, cleanupShim, err := resolveToolchain(universe.request)
	if err != nil {
		return err
	}
	defer cleanupShim()
	if toolchain != universe.toolchain {
		return invalidHydration(
			"selected toolchain changed between resolution and hydration",
		)
	}
	fset := token.NewFileSet()
	loaded, err := packages.Load(&packages.Config{
		Mode: hydrationMode,
		Dir:  universe.request.Dir,
		Fset: fset,
		ParseFile: func(
			fset *token.FileSet,
			filename string,
			src []byte,
		) (*ast.File, error) {
			return parser.ParseFile(
				fset,
				filename,
				src,
				parser.ParseComments|parser.SkipObjectResolution,
			)
		},
		Overlay:    universe.request.Overlay,
		BuildFlags: universe.request.BuildFlags,
		Env:        env,
	}, patterns...)
	if err != nil {
		return invalidHydration("checker load failed: %v", err)
	}
	semantic := map[string]*packages.Package{}
	var walkErr error
	packages.Visit(loaded, nil, func(pkg *packages.Package) {
		if walkErr != nil {
			return
		}
		if len(pkg.Errors) > 0 && pkg.PkgPath != "builtin" {
			walkErr = invalidHydration(
				"%s: %s", pkg.PkgPath, pkg.Errors[0],
			)
			return
		}
		if prior, duplicate := semantic[pkg.PkgPath]; duplicate &&
			prior != pkg {
			walkErr = invalidHydration(
				"checker returned duplicate package path %s",
				pkg.PkgPath,
			)
			return
		}
		semantic[pkg.PkgPath] = pkg
	})
	if walkErr != nil {
		return walkErr
	}
	for packageID, record := range initial {
		checked := semantic[record.id.ImportPath()]
		if checked == nil {
			return invalidHydration(
				"selected package %s is absent from checker load",
				packageID,
			)
		}
		if checked.Types == nil || checked.TypesInfo == nil {
			return invalidHydration(
				"selected package %s lacks checker evidence",
				packageID,
			)
		}
		if checked.Types.Path() != record.id.ImportPath() {
			return invalidHydration(
				"type path %s disagrees with %s",
				checked.Types.Path(), record.id,
			)
		}
		if checked.Name != record.name ||
			checked.Types.Name() != record.name {
			return invalidHydration(
				"package name %q/%q disagrees with %s name %q",
				checked.Name, checked.Types.Name(), record.id, record.name,
			)
		}
		record.types = checked.Types
		if err := attachHydratedPackage(
			universe,
			record,
			checked,
			fset,
			files,
			synthetic[packageID] != nil,
		); err != nil {
			return err
		}
	}
	universe.fset = fset
	universe.hydrated = true
	if err := universe.CheckTypeGraphCoherence(); err != nil {
		return err
	}
	return nil
}

func resolveHydrationDemand(
	universe *Universe,
	request HydrationRequest,
) (
	map[identity.FileID]bool,
	map[identity.PackageID]*LoadedPackage,
	error,
) {
	knownFiles := fileByID(universe)
	owners := packageForFile(universe)
	knownPackages := map[identity.PackageID]*LoadedPackage{}
	for _, pkg := range universe.packages {
		knownPackages[pkg.id] = pkg
	}
	files := map[identity.FileID]bool{}
	synthetic := map[identity.PackageID]*LoadedPackage{}
	for _, fileID := range request.files {
		file := knownFiles[fileID]
		if file == nil {
			return nil, nil, invalidHydration(
				"request names unknown file %s", fileID,
			)
		}
		pkg := owners[fileID]
		if pkg.disposition != DispositionOrdinarySource &&
			pkg.disposition != DispositionUnsafeIntrinsic {
			return nil, nil, invalidHydration(
				"file %s has non-structural disposition %s",
				fileID, pkg.disposition,
			)
		}
		files[fileID] = true
	}
	for _, packageID := range request.synthetic {
		pkg := knownPackages[packageID]
		if pkg == nil {
			return nil, nil, invalidHydration(
				"request names unknown synthetic package %s",
				packageID,
			)
		}
		if !pkg.hasCheckedView {
			return nil, nil, invalidHydration(
				"package %s has no checked view", packageID,
			)
		}
		synthetic[packageID] = pkg
	}
	for fileID := range files {
		pkg := owners[fileID]
		if knownFiles[fileID].cgoOriginal && synthetic[pkg.id] == nil {
			return nil, nil, invalidHydration(
				"local cgo file %s lacks its checked-view package",
				fileID,
			)
		}
	}
	return files, synthetic, nil
}

func attachHydratedPackage(
	universe *Universe,
	record *LoadedPackage,
	checked *packages.Package,
	fset *token.FileSet,
	localFiles map[identity.FileID]bool,
	localSynthetic bool,
) error {
	record.typesInfo = checked.TypesInfo
	syntaxByPath := map[string]*ast.File{}
	for index, syntax := range checked.Syntax {
		if index >= len(checked.CompiledGoFiles) {
			return invalidHydration(
				"%s has syntax without a compiled-file identity", record.id,
			)
		}
		syntaxByPath[filepath.Clean(
			checked.CompiledGoFiles[index],
		)] = syntax
	}
	sourcePaths := map[string]bool{}
	for _, file := range record.files {
		sourcePaths[filepath.Clean(file.path)] = true
		if syntax := syntaxByPath[filepath.Clean(file.path)]; syntax != nil {
			file.checkerFile = fset.File(syntax.Pos())
			if file.checkerFile == nil {
				return invalidHydration(
					"%s has no checker token file", file.id,
				)
			}
		}
	}
	if localSynthetic {
		record.checkedDecls = nil
		for path, syntax := range syntaxByPath {
			if sourcePaths[path] {
				continue
			}
			for _, declaration := range syntax.Decls {
				record.checkedDecls = append(
					record.checkedDecls,
					checkedDecl{node: declaration},
				)
			}
		}
		if len(record.checkedDecls) == 0 {
			return invalidHydration(
				"checked-view package %s has no generated declarations",
				record.id,
			)
		}
	}
	for _, file := range record.files {
		if !localFiles[file.id] {
			continue
		}
		raw, err := fileBytes(file.path, universe.request.Overlay)
		if err != nil {
			return invalidHydration(
				"%s bytes unreadable: %v", file.id, err,
			)
		}
		if sha256.Sum256(raw) != file.byteDigest {
			return invalidHydration(
				"%s bytes changed after resolution", file.id,
			)
		}
		file.selectedBytes = append([]byte(nil), raw...)
		syntax := syntaxByPath[filepath.Clean(file.path)]
		switch {
		case syntax != nil:
			file.fset = fset
			file.syntax = syntax
			file.physicalFset = fset
			file.physicalSyntax = syntax
			if version, present := checked.TypesInfo.FileVersions[syntax]; present && version != "" {
				if file.effectiveVersion != "" &&
					file.effectiveVersion != version {
					return invalidHydration(
						"%s version %q disagrees with checker %q",
						file.id, file.effectiveVersion, version,
					)
				}
				file.effectiveVersion = version
			}
		case file.cgoOriginal:
			physicalFset := token.NewFileSet()
			physical, parseErr := parser.ParseFile(
				physicalFset,
				file.path,
				raw,
				parser.ParseComments|parser.SkipObjectResolution,
			)
			if parseErr != nil {
				return invalidHydration(
					"%s cgo source unparsable: %v", file.id, parseErr,
				)
			}
			file.physicalFset = physicalFset
			file.physicalSyntax = physical
		case record.disposition == DispositionUnsafeIntrinsic:
			physicalFset := token.NewFileSet()
			physical, parseErr := parser.ParseFile(
				physicalFset,
				file.path,
				raw,
				parser.ParseComments|parser.SkipObjectResolution,
			)
			if parseErr != nil {
				return invalidHydration(
					"%s intrinsic source unparsable: %v",
					file.id, parseErr,
				)
			}
			file.physicalFset = physicalFset
			file.physicalSyntax = physical
		default:
			return invalidHydration(
				"%s has no checked syntax", file.id,
			)
		}
	}
	return nil
}

// HydrationStats exposes bounded lifecycle evidence without exposing syntax.
type HydrationStats struct {
	SemanticPackages int
	LocalFiles       int
	LocalBytes       int64
	CheckedPackages  int
}

// HydrationStats reports the exact selectively retained transient surface.
func (u *Universe) HydrationStats() HydrationStats {
	var out HydrationStats
	for _, pkg := range u.packages {
		if pkg.types != nil {
			out.SemanticPackages++
		}
		if len(pkg.checkedDecls) > 0 {
			out.CheckedPackages++
		}
		for _, file := range pkg.files {
			if file.physicalSyntax != nil {
				out.LocalFiles++
				out.LocalBytes += int64(len(file.selectedBytes))
			}
		}
	}
	return out
}
