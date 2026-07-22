package source

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/identity"
)

// loadMode is the single load configuration: requested roots AND their
// complete dependency closure resolve in one packages.Load invocation, so
// every package belongs to one coherent go/types object graph. There is no
// second load and no metadata-now/hydrate-later route.
const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedModule |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedEmbedFiles |
	packages.NeedEmbedPatterns

// LoadUniverse resolves one compilation request into the transient typed
// universe (one coherent go/types graph, complete unit census, cgo origin
// evidence). It fails closed on any package error, type error, identity
// collision, or unclassifiable package; there is no partial universe.
//
// The acquisition policy is contract-derived data resolved BEFORE this call;
// the loader itself makes no policy decision. The manifest supplies verified
// provider unit evidence for manifest-mode files; the zero manifest supplies
// nothing, so a manifest-mode file fails closed without a produced artifact.
// Evidence-depth selection and retention happen downstream in the scope phase
// and Finalize; the loader never performs retention by mutating records and
// never assigns depth.
func LoadUniverse(req Request, policy AcquisitionPolicy, manifest UnitManifest) (*Universe, error) {
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	toolchain, env, cleanupShim, err := resolveToolchain(req)
	if err != nil {
		return nil, err
	}
	defer cleanupShim()
	stdSet, err := listPatternSet(toolchain, env, req.Dir, "std")
	if err != nil {
		return nil, err
	}
	cmdSet, err := listPatternSet(toolchain, env, req.Dir, "cmd")
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: loadMode,
		Dir:  req.Dir,
		Fset: fset,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
		},
		Overlay:    req.Overlay,
		BuildFlags: req.BuildFlags,
		Env:        env,
	}
	loaded, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, &LoadError{Dir: req.Dir, Reason: "load failed: " + err.Error()}
	}
	requested := map[string]bool{}
	for _, pkg := range loaded {
		if len(pkg.GoFiles) > 0 || len(pkg.CompiledGoFiles) > 0 || pkg.PkgPath == "unsafe" {
			requested[pkg.PkgPath] = true
		}
	}
	if len(requested) == 0 {
		return nil, &LoadError{Dir: req.Dir, Reason: "no Go source packages matched " + strings.Join(patterns, " ")}
	}
	classifier := &classifier{
		req: req, policy: policy, toolchain: toolchain, stdSet: stdSet, cmdSet: cmdSet,
		moduleDirs: map[identity.ModuleID]string{},
	}
	universe := &Universe{fset: fset, toolchain: toolchain, request: req, manifest: manifest}
	seen := map[identity.PackageID]bool{}
	var walkErr error
	packages.Visit(loaded, nil, func(pkg *packages.Package) {
		if walkErr != nil {
			return
		}
		if len(pkg.Errors) > 0 && pkg.PkgPath != "builtin" {
			walkErr = &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": " + pkg.Errors[0].Error()}
			return
		}
		if len(pkg.GoFiles) == 0 && len(pkg.CompiledGoFiles) == 0 && pkg.PkgPath != "unsafe" {
			return
		}
		record, err := classifier.record(pkg, requested[pkg.PkgPath], fset)
		if err != nil {
			walkErr = err
			return
		}
		if seen[record.id] {
			walkErr = &LoadError{Dir: req.Dir, Reason: "duplicate package identity " + record.id.String()}
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
	if len(universe.roots) == 0 {
		return nil, &LoadError{Dir: req.Dir, Reason: "no requested roots after classification"}
	}
	sort.Slice(universe.packages, func(i, j int) bool { return universe.packages[i].id.String() < universe.packages[j].id.String() })
	sort.Slice(universe.roots, func(i, j int) bool { return universe.roots[i].id.String() < universe.roots[j].id.String() })
	if err := censusUniverse(universe); err != nil {
		return nil, err
	}
	return universe, nil
}

// resolveToolchain resolves the exact selected go binary and its fingerprint,
// and produces the environment shared by the loader and every toolchain
// execution. The go/packages driver resolves "go" from PATH; a shim directory
// whose "go" entry links to the exact selected binary guarantees the driver
// invokes it even when the selected filename is not "go".
func resolveToolchain(req Request) (Toolchain, []string, func(), error) {
	noop := func() {}
	binary := req.GoBinary
	if binary == "" {
		resolved, err := exec.LookPath("go")
		if err != nil {
			return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "no go binary selected or on PATH: " + err.Error()}
		}
		binary = resolved
	}
	absolute, err := filepath.Abs(binary)
	if err != nil {
		return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "toolchain path unresolvable: " + err.Error()}
	}
	shimDir, err := os.MkdirTemp("", "gotots-toolchain-*")
	if err != nil {
		return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "toolchain shim: " + err.Error()}
	}
	cleanup := func() { os.RemoveAll(shimDir) }
	if err := os.Symlink(absolute, filepath.Join(shimDir, "go")); err != nil {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "toolchain shim: " + err.Error()}
	}
	env := append(os.Environ(), req.Env...)
	env = append(env, "PATH="+shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := runGo(absolute, env, req.Dir, "env",
		"GOROOT", "GOVERSION", "GOOS", "GOARCH", "GOEXPERIMENT", "GOFLAGS", "CGO_ENABLED")
	if err != nil {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "toolchain fingerprint failed: " + err.Error()}
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 7 {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "toolchain fingerprint output malformed"}
	}
	binaryBytes, err := os.ReadFile(absolute)
	if err != nil {
		cleanup()
		return Toolchain{}, nil, noop, &LoadError{Dir: req.Dir, Reason: "toolchain binary unreadable: " + err.Error()}
	}
	return Toolchain{
		binary: absolute, binaryDigest: fmt.Sprintf("%x", sha256.Sum256(binaryBytes)),
		goroot: strings.TrimSpace(lines[0]), version: strings.TrimSpace(lines[1]),
		goos: strings.TrimSpace(lines[2]), goarch: strings.TrimSpace(lines[3]),
		experiments: strings.TrimSpace(lines[4]), goflags: strings.TrimSpace(lines[5]),
		cgoEnabled: strings.TrimSpace(lines[6]),
	}, env, cleanup, nil
}

// listPatternSet resolves one authoritative toolchain pattern set (std, cmd)
// under the selected binary.
func listPatternSet(toolchain Toolchain, env []string, dir, pattern string) (map[string]bool, error) {
	out, err := runGo(toolchain.binary, env, dir, "list", pattern)
	if err != nil {
		return nil, &LoadError{Dir: dir, Reason: "go list " + pattern + " failed: " + err.Error()}
	}
	set := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			set[line] = true
		}
	}
	return set, nil
}

// runGo executes the selected go binary.
func runGo(binary string, env []string, dir string, args ...string) (string, error) {
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", &LoadError{Dir: dir, Reason: strings.Join(args, " ") + ": " + err.Error() + ": " + stderr.String()}
	}
	return stdout.String(), nil
}

// classifier assigns each closure package its owner, provenance, acquisition,
// and language disposition from resolved toolchain facts. Import-path
// spelling, path prefixes, and Module==nil are never classification rules.
type classifier struct {
	req        Request
	policy     AcquisitionPolicy
	toolchain  Toolchain
	stdSet     map[string]bool
	cmdSet     map[string]bool
	moduleDirs map[identity.ModuleID]string
}

func (c *classifier) record(pkg *packages.Package, isRoot bool, fset *token.FileSet) (*LoadedPackage, error) {
	out := &LoadedPackage{requestedRoot: isRoot, disposition: DispositionOrdinarySource}
	var owner identity.Owner
	var relBase string
	switch {
	case pkg.PkgPath == "builtin":
		owner = identity.LanguagePseudoOwner()
		out.provenance, out.acquisition, out.disposition = ProvenanceLanguagePseudo, AcquisitionGOROOT, DispositionBuiltinUniverse
		relBase = filepath.Join(c.toolchain.goroot, "src")
	case c.stdSet[pkg.PkgPath]:
		owner = identity.StandardLibraryOwner()
		out.provenance, out.acquisition = ProvenanceStandardLibrary, AcquisitionGOROOT
		if pkg.PkgPath == "unsafe" {
			out.disposition = DispositionUnsafeIntrinsic
		}
		relBase = filepath.Join(c.toolchain.goroot, "src")
	case c.cmdSet[pkg.PkgPath]:
		// cmd-set membership is authoritative and includes the toolchain's
		// internal libraries; no path-prefix rule participates.
		owner = identity.ToolchainOwner()
		out.provenance, out.acquisition = ProvenanceToolchainPackage, AcquisitionGOROOT
		relBase = filepath.Join(c.toolchain.goroot, "src")
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
			return nil, err
		}
		if prior, dup := c.moduleDirs[moduleID]; dup && prior != moduleDir && moduleDir != "" && prior != "" {
			return nil, &LoadError{Dir: c.req.Dir, Reason: "module identity collision: " + moduleID.String() +
				" resolves to both " + prior + " and " + moduleDir}
		}
		if moduleDir != "" {
			c.moduleDirs[moduleID] = moduleDir
		}
		owner, err = identity.NewModuleOwner(moduleID)
		if err != nil {
			return nil, err
		}
		out.moduleGoVersion = pkg.Module.GoVersion
		if pkg.Module.Main {
			out.provenance, out.acquisition = ProvenanceWorkspaceModule, AcquisitionWorkspace
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
		relBase = moduleDir
		if relBase == "" {
			relBase = vendorBase(pkg, c.req.Dir)
		}
	default:
		return nil, &LoadError{Dir: c.req.Dir, Reason: pkg.PkgPath +
			": package is neither module-owned nor in the selected toolchain's std/cmd sets"}
	}
	packageID, err := identity.NewPackageID(owner, pkg.PkgPath)
	if err != nil {
		return nil, err
	}
	out.id = packageID
	out.imports = importPaths(pkg)
	if err := c.attachInputs(out, pkg, owner, relBase, fset); err != nil {
		return nil, err
	}
	return out, nil
}

// attachInputs models the package's complete inputs: identity-bearing source
// Go files with their checked syntax, the checked-view difference for cgo
// packages, non-Go inputs, and embed files/patterns. The loader never aborts a
// cgo-importing package; the view difference is recorded and later semantic
// planning assigns per-body external obligations.
func (c *classifier) attachInputs(out *LoadedPackage, pkg *packages.Package, owner identity.Owner, relBase string, fset *token.FileSet) error {
	if pkg.PkgPath == "builtin" {
		for _, goFile := range pkg.GoFiles {
			fileID, err := c.fileIDFor(owner, relBase, goFile)
			if err != nil {
				return err
			}
			out.files = append(out.files, &LoadedFile{path: goFile, id: fileID})
		}
		sortLoadedFiles(out.files)
		return nil
	}
	out.types, out.typesInfo = pkg.Types, pkg.TypesInfo
	// Checked syntax aligns with CompiledGoFiles. Compiled files under the
	// owner root are ordinary source; compiled files outside it are the cgo
	// checked view.
	syntaxByPath := map[string]*ast.File{}
	for i, syntax := range pkg.Syntax {
		if i < len(pkg.CompiledGoFiles) {
			syntaxByPath[pkg.CompiledGoFiles[i]] = syntax
		}
	}
	cgoTransformed := false
	for _, compiled := range pkg.CompiledGoFiles {
		if rel, err := filepath.Rel(relBase, compiled); err != nil || strings.HasPrefix(rel, "..") {
			cgoTransformed = true
			// Checked-view decls are joined to origins at census time via
			// //line evidence; the checked path is transient acquisition data.
			if syntax := syntaxByPath[compiled]; syntax != nil {
				for _, decl := range syntax.Decls {
					out.checkedDecls = append(out.checkedDecls, checkedDecl{node: decl, fromFile: compiled})
				}
			}
		}
	}
	for _, goFile := range pkg.GoFiles {
		fileID, err := c.fileIDFor(owner, relBase, goFile)
		if err != nil {
			return err
		}
		file := &LoadedFile{path: goFile, id: fileID}
		mode, err := c.policy.ModeFor(out.id)
		if err != nil {
			return err
		}
		file.censusMode = mode
		if _, isOverlaid := c.req.Overlay[goFile]; isOverlaid {
			file.overlaid = true
		}
		if syntax, checked := syntaxByPath[goFile]; checked {
			file.fset = fset
			file.syntax = syntax
			if pkg.TypesInfo != nil {
				file.effectiveVersion = pkg.TypesInfo.FileVersions[syntax]
			}
			if file.effectiveVersion == "" && pkg.Module != nil && pkg.Module.GoVersion != "" {
				file.effectiveVersion = "go" + pkg.Module.GoVersion
			}
		} else if out.disposition == DispositionOrdinarySource && !cgoTransformed {
			// A source file may lack checked syntax only when its checked
			// form genuinely lives in a differing checked view (cgo). A
			// missing-syntax file without that corroboration is a defect,
			// never a silent metadata downgrade.
			return &LoadError{Dir: c.req.Dir, Reason: fileID.String() +
				": file has no checked syntax and the package has no checked-view difference"}
		} else {
			file.cgoOriginal = true
		}
		out.files = append(out.files, file)
	}
	sortLoadedFiles(out.files)
	for _, other := range pkg.OtherFiles {
		out.otherFiles = append(out.otherFiles, c.ownerRel(relBase, other))
	}
	sort.Strings(out.otherFiles)
	for _, embed := range pkg.EmbedFiles {
		out.embedFiles = append(out.embedFiles, c.ownerRel(relBase, embed))
	}
	sort.Strings(out.embedFiles)
	// x/tools absolutizes embed patterns; restore the declared package-dir-
	// relative spelling.
	for _, pattern := range pkg.EmbedPatterns {
		out.embedPatterns = append(out.embedPatterns, c.ownerRel(pkg.Dir, pattern))
	}
	sort.Strings(out.embedPatterns)
	return nil
}

// ownerRel renders an input path owner-relative when possible, keeping the
// raw path otherwise (display only; never an identity).
func (c *classifier) ownerRel(relBase, path string) string {
	if rel, err := filepath.Rel(relBase, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return path
}

// fileIDFor derives one file's owner-relative identity.
func (c *classifier) fileIDFor(owner identity.Owner, relBase, osPath string) (identity.FileID, error) {
	rel, err := filepath.Rel(relBase, osPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return identity.FileID{}, &LoadError{Dir: c.req.Dir, Reason: osPath + ": file lies outside its owner root " + relBase}
	}
	return identity.NewFileID(owner, filepath.ToSlash(rel))
}

func sortFiles(files []*File) {
	sort.Slice(files, func(i, j int) bool { return files[i].id.Rel() < files[j].id.Rel() })
}

// vendoredDir reports whether the package's files live under the workspace
// vendor tree (an acquisition fact derived from resolved build inputs).
func vendoredDir(pkg *packages.Package, workspaceDir string) bool {
	if len(pkg.GoFiles) == 0 {
		return false
	}
	rel, err := filepath.Rel(workspaceDir, pkg.GoFiles[0])
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return strings.HasPrefix(rel, "vendor/") || strings.Contains(rel, "/vendor/")
}

// vendorBase resolves the identity base of a vendored module: the vendor
// subtree named by the module path, so vendored files keep module-relative
// identities.
func vendorBase(pkg *packages.Package, workspaceDir string) string {
	if len(pkg.GoFiles) == 0 || pkg.Module == nil {
		return workspaceDir
	}
	dir := filepath.Dir(pkg.GoFiles[0])
	marker := filepath.FromSlash("vendor/" + pkg.Module.Path)
	if idx := strings.Index(dir, marker); idx >= 0 {
		return dir[:idx+len(marker)]
	}
	return workspaceDir
}

// importPaths lists the package's imports, sorted.
func importPaths(pkg *packages.Package) []string {
	out := make([]string, 0, len(pkg.Imports))
	for path := range pkg.Imports {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}
