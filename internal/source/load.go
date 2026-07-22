package source

import (
	"bytes"
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

// rootMode loads the requested roots with syntax and complete type
// information (imports satisfied from export data).
const rootMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedModule |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo

// closureMode resolves the complete transitive package closure metadata.
const closureMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedModule

// LoadWorkspace resolves one compilation request into the typed universe. It
// fails closed on any package error, type error, identity collision, or
// unclassifiable package; there is no partial universe.
func LoadWorkspace(req Request) (*Workspace, error) {
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	toolchain, env, err := resolveToolchain(req)
	if err != nil {
		return nil, err
	}
	stdSet, err := listPatternSet(toolchain, env, req.Dir, "std")
	if err != nil {
		return nil, err
	}
	cmdSet, err := listPatternSet(toolchain, env, req.Dir, "cmd")
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	baseConfig := func(mode packages.LoadMode) *packages.Config {
		return &packages.Config{
			Mode: mode,
			Dir:  req.Dir,
			Fset: fset,
			ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
				return parser.ParseFile(fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
			},
			Overlay:    req.Overlay,
			BuildFlags: req.BuildFlags,
			Env:        env,
		}
	}
	roots, err := packages.Load(baseConfig(rootMode), patterns...)
	if err != nil {
		return nil, &LoadError{Dir: req.Dir, Reason: "root load failed: " + err.Error()}
	}
	closure, err := packages.Load(baseConfig(closureMode), patterns...)
	if err != nil {
		return nil, &LoadError{Dir: req.Dir, Reason: "closure load failed: " + err.Error()}
	}
	rootByPath := map[string]*packages.Package{}
	for _, pkg := range roots {
		if pkg.PkgPath == "builtin" {
			// The builtin pseudo-package redeclares the predeclared universe
			// and can never be type-checked as ordinary source; it joins the
			// closure as a language-pseudo metadata record only.
			continue
		}
		if len(pkg.Errors) > 0 {
			return nil, &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": " + pkg.Errors[0].Error()}
		}
		if len(pkg.CompiledGoFiles) == 0 && len(pkg.GoFiles) == 0 {
			continue
		}
		rootByPath[pkg.PkgPath] = pkg
	}
	if len(rootByPath) == 0 {
		return nil, &LoadError{Dir: req.Dir, Reason: "no Go source packages matched " + strings.Join(patterns, " ")}
	}
	classifier := &classifier{
		req: req, toolchain: toolchain, stdSet: stdSet, cmdSet: cmdSet,
		moduleDirs: map[identity.ModuleID]string{},
	}
	ws := &Workspace{fset: fset, toolchain: toolchain}
	seen := map[identity.PackageID]bool{}
	var walkErr error
	packages.Visit(closure, nil, func(pkg *packages.Package) {
		if walkErr != nil {
			return
		}
		if len(pkg.Errors) > 0 {
			walkErr = &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": " + pkg.Errors[0].Error()}
			return
		}
		if len(pkg.GoFiles) == 0 && len(pkg.CompiledGoFiles) == 0 && pkg.PkgPath != "unsafe" {
			return
		}
		root := rootByPath[pkg.PkgPath]
		record, err := classifier.record(pkg, root, fset)
		if err != nil {
			walkErr = err
			return
		}
		if seen[record.id] {
			walkErr = &LoadError{Dir: req.Dir, Reason: "duplicate package identity " + record.id.String()}
			return
		}
		seen[record.id] = true
		ws.packages = append(ws.packages, record)
		if record.selected {
			ws.selected = append(ws.selected, record)
		}
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if len(ws.selected) == 0 {
		return nil, &LoadError{Dir: req.Dir, Reason: "no selected packages after classification"}
	}
	sort.Slice(ws.packages, func(i, j int) bool { return ws.packages[i].id.String() < ws.packages[j].id.String() })
	sort.Slice(ws.selected, func(i, j int) bool { return ws.selected[i].id.String() < ws.selected[j].id.String() })
	return ws, nil
}

// resolveToolchain resolves the exact selected go binary and its fingerprint,
// and produces the environment (with PATH pinned to that binary) shared by the
// loader and every downstream toolchain execution.
func resolveToolchain(req Request) (Toolchain, []string, error) {
	binary := req.GoBinary
	if binary == "" {
		resolved, err := exec.LookPath("go")
		if err != nil {
			return Toolchain{}, nil, &LoadError{Dir: req.Dir, Reason: "no go binary selected or on PATH: " + err.Error()}
		}
		binary = resolved
	}
	absolute, err := filepath.Abs(binary)
	if err != nil {
		return Toolchain{}, nil, &LoadError{Dir: req.Dir, Reason: "toolchain path unresolvable: " + err.Error()}
	}
	env := append(os.Environ(), req.Env...)
	env = append(env, "PATH="+filepath.Dir(absolute)+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := runGo(absolute, env, req.Dir, "env", "GOROOT", "GOVERSION")
	if err != nil {
		return Toolchain{}, nil, &LoadError{Dir: req.Dir, Reason: "toolchain fingerprint failed: " + err.Error()}
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		return Toolchain{}, nil, &LoadError{Dir: req.Dir, Reason: "toolchain fingerprint output malformed"}
	}
	return Toolchain{binary: absolute, goroot: strings.TrimSpace(lines[0]), version: strings.TrimSpace(lines[1])}, env, nil
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
	toolchain  Toolchain
	stdSet     map[string]bool
	cmdSet     map[string]bool
	moduleDirs map[identity.ModuleID]string
}

func (c *classifier) record(pkg *packages.Package, root *packages.Package, fset *token.FileSet) (*Package, error) {
	out := &Package{selected: root != nil}
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
	case c.cmdSet[pkg.PkgPath] || strings.HasPrefix(pkg.PkgPath, "cmd/") && pkg.Module == nil:
		// cmd-set membership is authoritative; internal toolchain packages
		// reachable only from cmd roots share the toolchain owner.
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
			// Vendored modules report no module directory; files live under
			// the workspace vendor tree and stay identified module-relative
			// through the vendor path suffix.
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
	if root != nil {
		if err := c.attachSelected(out, root, owner, relBase, fset); err != nil {
			return nil, err
		}
		return out, nil
	}
	for _, goFile := range pkg.GoFiles {
		fileID, err := c.fileIDFor(owner, relBase, goFile, pkg)
		if err != nil {
			return nil, err
		}
		out.files = append(out.files, &File{path: goFile, id: fileID})
	}
	sort.Slice(out.files, func(i, j int) bool { return out.files[i].id.Rel() < out.files[j].id.Rel() })
	return out, nil
}

// attachSelected joins the syntax/type-bearing root package onto the record.
func (c *classifier) attachSelected(out *Package, root *packages.Package, owner identity.Owner, relBase string, fset *token.FileSet) error {
	if root.Types == nil || root.TypesInfo == nil {
		return &LoadError{Dir: c.req.Dir, Reason: root.PkgPath + ": type information missing"}
	}
	out.types, out.typesInfo = root.Types, root.TypesInfo
	if len(root.Syntax) != len(root.CompiledGoFiles) {
		return &CgoUnsupportedError{ImportPath: root.PkgPath}
	}
	for i, syntax := range root.Syntax {
		osPath := root.CompiledGoFiles[i]
		fileID, err := c.fileIDFor(owner, relBase, osPath, root)
		if err != nil {
			// A compiled file outside the owner root is cgo/generated
			// intermediate output: an explicit disposition.
			return &CgoUnsupportedError{ImportPath: root.PkgPath}
		}
		effective := root.TypesInfo.FileVersions[syntax]
		if effective == "" && root.Module != nil {
			effective = "go" + root.Module.GoVersion
		}
		out.files = append(out.files, &File{
			path: osPath, id: fileID, fset: fset, syntax: syntax, effectiveVersion: effective,
		})
	}
	sort.Slice(out.files, func(i, j int) bool { return out.files[i].id.Rel() < out.files[j].id.Rel() })
	return nil
}

// fileIDFor derives one file's owner-relative identity.
func (c *classifier) fileIDFor(owner identity.Owner, relBase, osPath string, pkg *packages.Package) (identity.FileID, error) {
	rel, err := filepath.Rel(relBase, osPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return identity.FileID{}, &LoadError{Dir: c.req.Dir, Reason: osPath + ": file lies outside its owner root " + relBase}
	}
	return identity.NewFileID(owner, filepath.ToSlash(rel))
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
