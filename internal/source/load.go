package source

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/identity"
)

// resolutionMode acquires the complete import closure and physical inputs
// without parsing or typechecking dependency interiors. It is the sole
// pre-planning load.
const resolutionMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedModule |
	packages.NeedEmbedFiles |
	packages.NeedEmbedPatterns

// ResolveUniverse resolves one compilation request into a complete immutable-
// identity metadata closure. It intentionally produces no AST or checker
// graph: structural-source planning must run before semantic hydration.
func ResolveUniverse(req Request) (*Universe, error) {
	req = cloneRequest(req)
	patterns := requestPatterns(req)
	toolchain, env, cleanupShim, err := resolveToolchain(req)
	if err != nil {
		return nil, err
	}
	defer cleanupShim()
	req.GoBinary = toolchain.binary
	stdSet, err := listPatternSet(toolchain, env, req.Dir, "std")
	if err != nil {
		return nil, err
	}
	cmdSet, err := listPatternSet(toolchain, env, req.Dir, "cmd")
	if err != nil {
		return nil, err
	}
	loaded, err := packages.Load(&packages.Config{
		Mode:       resolutionMode,
		Dir:        req.Dir,
		Overlay:    req.Overlay,
		BuildFlags: req.BuildFlags,
		Env:        env,
	}, patterns...)
	if err != nil {
		return nil, &LoadError{
			Dir: req.Dir, Reason: "metadata resolution failed: " + err.Error(),
		}
	}
	requested := map[string]bool{}
	for _, pkg := range loaded {
		if packageHasInputs(pkg) {
			requested[pkg.PkgPath] = true
		}
	}
	if len(requested) == 0 {
		return nil, &LoadError{
			Dir: req.Dir,
			Reason: "no Go source packages matched " +
				strings.Join(patterns, " "),
		}
	}
	classifier := &classifier{
		req: req, toolchain: toolchain, stdSet: stdSet, cmdSet: cmdSet,
		moduleDirs: map[identity.ModuleID]string{},
		gorootGo:   readGoDirective(filepath.Join(toolchain.goroot, "src", "go.mod")),
	}
	universe := &Universe{toolchain: toolchain, request: req}
	seen := map[identity.PackageID]bool{}
	var walkErr error
	packages.Visit(loaded, nil, func(pkg *packages.Package) {
		if walkErr != nil {
			return
		}
		if len(pkg.Errors) > 0 && pkg.PkgPath != "builtin" {
			walkErr = &LoadError{
				Dir:    req.Dir,
				Reason: pkg.PkgPath + ": " + pkg.Errors[0].Error(),
			}
			return
		}
		if !packageHasInputs(pkg) {
			return
		}
		record, recordErr := classifier.record(
			pkg, requested[pkg.PkgPath],
		)
		if recordErr != nil {
			walkErr = recordErr
			return
		}
		if seen[record.id] {
			walkErr = &LoadError{
				Dir:    req.Dir,
				Reason: "duplicate package identity " + record.id.String(),
			}
			return
		}
		seen[record.id] = true
		universe.packages = append(universe.packages, record)
		if record.requestedRoot {
			universe.roots = append(universe.roots, record)
		}
	})
	if walkErr != nil {
		return nil, walkErr
	}
	builtinID, err := identity.NewPackageID(
		identity.LanguagePseudoOwner(), "builtin",
	)
	if err != nil {
		return nil, err
	}
	if seen[builtinID] {
		return nil, &LoadError{
			Dir: req.Dir,
			Reason: "selected toolchain returned the language-owned " +
				"builtin universe as an ordinary package",
		}
	}
	universe.packages = append(universe.packages, &LoadedPackage{
		id:          builtinID,
		provenance:  ProvenanceLanguagePseudo,
		acquisition: AcquisitionGOROOT,
		disposition: DispositionBuiltinUniverse,
	})
	if len(universe.roots) == 0 {
		return nil, &LoadError{
			Dir: req.Dir, Reason: "no requested roots after classification",
		}
	}
	sort.Slice(universe.packages, func(i, j int) bool {
		return universe.packages[i].id.String() <
			universe.packages[j].id.String()
	})
	sort.Slice(universe.roots, func(i, j int) bool {
		return universe.roots[i].id.String() <
			universe.roots[j].id.String()
	})
	return universe, nil
}

func requestPatterns(req Request) []string {
	if len(req.Patterns) == 0 {
		return []string{"./..."}
	}
	return append([]string(nil), req.Patterns...)
}

func cloneRequest(req Request) Request {
	req.Patterns = append([]string(nil), req.Patterns...)
	req.BuildFlags = append([]string(nil), req.BuildFlags...)
	req.Env = append([]string(nil), req.Env...)
	if req.Overlay != nil {
		overlay := make(map[string][]byte, len(req.Overlay))
		for path, content := range req.Overlay {
			overlay[path] = append([]byte(nil), content...)
		}
		req.Overlay = overlay
	}
	return req
}

func packageHasInputs(pkg *packages.Package) bool {
	return len(pkg.GoFiles) > 0 ||
		len(pkg.CompiledGoFiles) > 0 ||
		pkg.PkgPath == "unsafe"
}

// classifier assigns each closure package its owner, provenance, acquisition,
// and language disposition from resolved toolchain facts. Import-path
// spelling, path prefixes, and Module==nil are never classification rules.
type classifier struct {
	req        Request
	toolchain  Toolchain
	stdSet     map[string]bool
	cmdSet     map[string]bool
	moduleDirs map[identity.ModuleID]string
	gorootGo   string
}

func (c *classifier) record(
	pkg *packages.Package,
	isRoot bool,
) (*LoadedPackage, error) {
	out := &LoadedPackage{
		requestedRoot: isRoot,
		disposition:   DispositionOrdinarySource,
	}
	owner, relBase, err := c.classifyPackage(pkg, out)
	if err != nil {
		return nil, err
	}
	packageID, err := identity.NewPackageID(owner, pkg.PkgPath)
	if err != nil {
		return nil, err
	}
	out.id = packageID
	out.imports = importPaths(pkg)
	if err := c.attachMetadata(out, pkg, owner, relBase); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *classifier) classifyPackage(
	pkg *packages.Package,
	out *LoadedPackage,
) (identity.Owner, string, error) {
	switch {
	case pkg.PkgPath == "builtin":
		out.provenance = ProvenanceLanguagePseudo
		out.acquisition = AcquisitionGOROOT
		out.disposition = DispositionBuiltinUniverse
		return identity.LanguagePseudoOwner(),
			filepath.Join(c.toolchain.goroot, "src"), nil
	case c.stdSet[pkg.PkgPath]:
		out.provenance = ProvenanceStandardLibrary
		out.acquisition = AcquisitionGOROOT
		if pkg.PkgPath == "unsafe" {
			out.disposition = DispositionUnsafeIntrinsic
		}
		return identity.StandardLibraryOwner(),
			filepath.Join(c.toolchain.goroot, "src"), nil
	case c.cmdSet[pkg.PkgPath]:
		out.provenance = ProvenanceToolchainPackage
		out.acquisition = AcquisitionGOROOT
		return identity.ToolchainOwner(),
			filepath.Join(c.toolchain.goroot, "src"), nil
	case pkg.Module != nil:
		return c.classifyModule(pkg, out)
	default:
		return identity.Owner{}, "", &LoadError{
			Dir: c.req.Dir,
			Reason: pkg.PkgPath +
				": package is neither module-owned nor in the selected " +
				"toolchain's std/cmd sets",
		}
	}
}

func (c *classifier) classifyModule(
	pkg *packages.Package,
	out *LoadedPackage,
) (identity.Owner, string, error) {
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
		return identity.Owner{}, "", err
	}
	if prior, duplicate := c.moduleDirs[moduleID]; duplicate &&
		prior != moduleDir && moduleDir != "" && prior != "" {
		return identity.Owner{}, "", &LoadError{
			Dir: c.req.Dir,
			Reason: "module identity collision: " + moduleID.String() +
				" resolves to both " + prior + " and " + moduleDir,
		}
	}
	if moduleDir != "" {
		c.moduleDirs[moduleID] = moduleDir
	}
	owner, err := identity.NewModuleOwner(moduleID)
	if err != nil {
		return identity.Owner{}, "", err
	}
	out.moduleGoVersion = pkg.Module.GoVersion
	if pkg.Module.Main {
		out.provenance = ProvenanceWorkspaceModule
		out.acquisition = AcquisitionWorkspace
	} else {
		out.provenance = ProvenanceModuleDependency
		switch {
		case pkg.Module.Replace != nil && pkg.Module.Replace.Version == "":
			out.acquisition = AcquisitionLocalReplacement
		case moduleDir == "" || vendoredDir(pkg, c.req.Dir):
			out.acquisition = AcquisitionVendor
		default:
			out.acquisition = AcquisitionModuleCache
		}
	}
	if moduleDir == "" {
		moduleDir = vendorBase(pkg, c.req.Dir)
	}
	return owner, moduleDir, nil
}

func (c *classifier) attachMetadata(
	out *LoadedPackage,
	pkg *packages.Package,
	owner identity.Owner,
	relBase string,
) error {
	sourcePaths := map[string]bool{}
	compiledPaths := map[string]bool{}
	for _, path := range pkg.GoFiles {
		sourcePaths[filepath.Clean(path)] = true
	}
	for _, path := range pkg.CompiledGoFiles {
		clean := filepath.Clean(path)
		compiledPaths[clean] = true
		if out.disposition == DispositionOrdinarySource &&
			!sourcePaths[clean] {
			out.hasCheckedView = true
		}
	}
	if out.disposition == DispositionOrdinarySource {
		for sourcePath := range sourcePaths {
			if !compiledPaths[sourcePath] {
				out.hasCheckedView = true
			}
		}
	}
	for _, goFile := range pkg.GoFiles {
		fileID, err := c.fileIDFor(owner, relBase, goFile)
		if err != nil {
			return err
		}
		raw, err := fileBytes(goFile, c.req.Overlay)
		if err != nil {
			return &LoadError{
				Dir: c.req.Dir,
				Reason: fileID.String() +
					": selected source bytes unreadable: " + err.Error(),
			}
		}
		version, err := effectiveGoVersion(
			raw, c.baseGoVersion(out),
		)
		if err != nil {
			return &LoadError{
				Dir: c.req.Dir,
				Reason: fileID.String() +
					": effective Go version: " + err.Error(),
			}
		}
		_, overlaid := c.req.Overlay[goFile]
		out.files = append(out.files, &LoadedFile{
			path: goFile, id: fileID,
			effectiveVersion: version,
			overlaid:         overlaid,
			cgoOriginal: out.disposition ==
				DispositionOrdinarySource &&
				out.hasCheckedView &&
				!compiledPaths[filepath.Clean(goFile)],
			byteDigest: sha256.Sum256(raw),
		})
	}
	sortLoadedFiles(out.files)
	for _, other := range pkg.OtherFiles {
		kind, err := inputKindForPath(other)
		if err != nil {
			return &LoadError{
				Dir:    c.req.Dir,
				Reason: out.id.String() + ": " + err.Error(),
			}
		}
		if err := c.attachInput(
			out, owner, relBase, pkg.Dir, other, kind,
		); err != nil {
			return err
		}
	}
	for _, embed := range pkg.EmbedFiles {
		if err := c.attachInput(
			out, owner, relBase, pkg.Dir, embed, InputEmbed,
		); err != nil {
			return err
		}
	}
	sort.Slice(out.inputs, func(i, j int) bool {
		return out.inputs[i].id.String() <
			out.inputs[j].id.String()
	})
	for index := 1; index < len(out.inputs); index++ {
		if out.inputs[index-1].id == out.inputs[index].id {
			return &LoadError{
				Dir: c.req.Dir,
				Reason: out.id.String() +
					": supplemental input has multiple semantic classes " +
					out.inputs[index].id.String(),
			}
		}
	}
	for _, pattern := range pkg.EmbedPatterns {
		out.embedPatterns = append(
			out.embedPatterns, packageRelative(pkg.Dir, pattern),
		)
	}
	sort.Strings(out.embedPatterns)
	return nil
}

func packageRelative(base, path string) string {
	if rel, err := filepath.Rel(base, path); err == nil &&
		!pathEscapes(rel) {
		return filepath.ToSlash(rel)
	}
	return path
}

func (c *classifier) attachInput(
	out *LoadedPackage,
	owner identity.Owner,
	relBase string,
	packageDir string,
	path string,
	kind InputKind,
) error {
	if !filepath.IsAbs(path) {
		path = filepath.Join(packageDir, path)
	}
	id, err := c.fileIDFor(owner, relBase, path)
	if err != nil {
		return err
	}
	raw, err := fileBytes(path, c.req.Overlay)
	if err != nil {
		return &LoadError{
			Dir: c.req.Dir,
			Reason: id.String() +
				": supplemental input unreadable: " + err.Error(),
		}
	}
	_, overlaid := c.req.Overlay[path]
	out.inputs = append(out.inputs, loadedInput{
		path: path, id: id, kind: kind,
		byteDigest: sha256.Sum256(raw), overlaid: overlaid,
	})
	return nil
}

func inputKindForPath(path string) (InputKind, error) {
	switch filepath.Ext(path) {
	case ".c":
		return InputC, nil
	case ".cc", ".cpp", ".cxx":
		return InputCXX, nil
	case ".m":
		return InputObjectiveC, nil
	case ".h", ".hh", ".hpp", ".hxx":
		return InputHeader, nil
	case ".f", ".F", ".for", ".f90":
		return InputFortran, nil
	case ".s", ".S", ".sx":
		return InputAssembly, nil
	case ".swig":
		return InputSWIG, nil
	case ".swigcxx":
		return InputSWIGCXX, nil
	case ".syso":
		return InputSyso, nil
	default:
		return InputInvalid, fmt.Errorf(
			"selected toolchain input %s has no closed input kind",
			path,
		)
	}
}

func (c *classifier) baseGoVersion(pkg *LoadedPackage) string {
	if pkg.moduleGoVersion != "" {
		return "go" + pkg.moduleGoVersion
	}
	switch pkg.id.Owner().Class() {
	case identity.OwnerStandardLibrary, identity.OwnerToolchain:
		return c.gorootGo
	default:
		return ""
	}
}

func fileBytes(path string, overlay map[string][]byte) ([]byte, error) {
	if content, overlaid := overlay[path]; overlaid {
		return append([]byte(nil), content...), nil
	}
	return os.ReadFile(path)
}

func sortLoadedFiles(files []*LoadedFile) {
	sort.Slice(files, func(left, right int) bool {
		return files[left].id.Rel() < files[right].id.Rel()
	})
}

func (c *classifier) fileIDFor(
	owner identity.Owner,
	relBase string,
	osPath string,
) (identity.FileID, error) {
	rel, err := filepath.Rel(relBase, osPath)
	if err != nil || pathEscapes(rel) {
		return identity.FileID{}, &LoadError{
			Dir: c.req.Dir,
			Reason: osPath +
				": file lies outside its owner root " + relBase,
		}
	}
	return identity.NewFileID(owner, filepath.ToSlash(rel))
}

func pathEscapes(rel string) bool {
	return rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func sortFiles(files []*File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].id.Rel() < files[j].id.Rel()
	})
}

func vendoredDir(pkg *packages.Package, workspaceDir string) bool {
	if len(pkg.GoFiles) == 0 {
		return false
	}
	rel, err := filepath.Rel(workspaceDir, pkg.GoFiles[0])
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "vendor/") ||
		strings.Contains(rel, "/vendor/")
}

func vendorBase(pkg *packages.Package, workspaceDir string) string {
	if len(pkg.GoFiles) == 0 || pkg.Module == nil {
		return workspaceDir
	}
	dir := filepath.Dir(pkg.GoFiles[0])
	marker := filepath.FromSlash("vendor/" + pkg.Module.Path)
	if index := strings.Index(dir, marker); index >= 0 {
		return dir[:index+len(marker)]
	}
	return workspaceDir
}

func importPaths(pkg *packages.Package) []string {
	out := make([]string, 0, len(pkg.Imports))
	for path := range pkg.Imports {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func fileByID(
	universe *Universe,
) map[identity.FileID]*LoadedFile {
	out := map[identity.FileID]*LoadedFile{}
	for _, pkg := range universe.packages {
		for _, file := range pkg.files {
			out[file.id] = file
		}
	}
	return out
}

func packageForFile(
	universe *Universe,
) map[identity.FileID]*LoadedPackage {
	out := map[identity.FileID]*LoadedPackage{}
	for _, pkg := range universe.packages {
		for _, file := range pkg.files {
			out[file.id] = pkg
		}
	}
	return out
}

func invalidHydration(reason string, args ...any) error {
	return &LoadError{
		Reason: "semantic hydration: " + fmt.Sprintf(reason, args...),
	}
}
