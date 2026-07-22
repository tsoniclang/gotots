package source

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/identity"
)

// loadMode requests names, files, syntax, modules, and complete type
// information for the selected packages (imports satisfied from export data).
const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedModule |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo

// LoadWorkspace resolves one compilation request into a typed workspace. It
// fails closed on any package error, type error, missing module, module
// identity collision, or file outside its module root; there is no partial
// universe.
func LoadWorkspace(req Request) (*Workspace, error) {
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
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
	}
	if len(req.Env) > 0 {
		cfg.Env = append(os.Environ(), req.Env...)
	}
	loaded, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, &LoadError{Dir: req.Dir, Reason: "package load failed: " + err.Error()}
	}
	if len(loaded) == 0 {
		return nil, &LoadError{Dir: req.Dir, Reason: "no packages matched " + strings.Join(patterns, " ")}
	}
	// Modules join by semantic identity; the same identity resolving to two
	// directories is an ambiguity, never silently tolerated.
	moduleDirs := map[identity.ModuleID]string{}
	ws := &Workspace{fset: fset}
	for _, pkg := range loaded {
		if len(pkg.Errors) > 0 {
			return nil, &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": " + pkg.Errors[0].Error()}
		}
		if len(pkg.Syntax) == 0 && len(pkg.CompiledGoFiles) == 0 {
			continue // empty match (e.g. a directory with no Go files)
		}
		if pkg.Module == nil {
			return nil, &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": package has no module; module-less loads are unsupported"}
		}
		moduleDir := pkg.Module.Dir
		if pkg.Module.Replace != nil {
			moduleDir = pkg.Module.Replace.Dir
		}
		if moduleDir == "" {
			return nil, &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": module " + pkg.Module.Path + " has no directory"}
		}
		version := pkg.Module.Version
		if pkg.Module.Main {
			version = ""
		}
		moduleID, err := identity.NewModuleID(pkg.Module.Path, version)
		if err != nil {
			return nil, err
		}
		if prior, seen := moduleDirs[moduleID]; seen && prior != moduleDir {
			return nil, &LoadError{Dir: req.Dir, Reason: "module identity collision: " + moduleID.String() +
				" resolves to both " + prior + " and " + moduleDir}
		}
		moduleDirs[moduleID] = moduleDir
		packageID, err := identity.NewPackageID(moduleID, pkg.PkgPath)
		if err != nil {
			return nil, err
		}
		if ws.goVersion < pkg.Module.GoVersion {
			ws.goVersion = pkg.Module.GoVersion
		}
		out := &Package{id: packageID, types: pkg.Types, typesInfo: pkg.TypesInfo}
		if pkg.TypesInfo == nil || pkg.Types == nil {
			return nil, &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": type information missing"}
		}
		if len(pkg.Syntax) != len(pkg.CompiledGoFiles) {
			return nil, &LoadError{Dir: req.Dir, Reason: pkg.PkgPath + ": syntax/file alignment mismatch"}
		}
		for i, syntax := range pkg.Syntax {
			osPath := pkg.CompiledGoFiles[i]
			rel, err := filepath.Rel(moduleDir, osPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				return nil, &LoadError{Dir: req.Dir, Reason: osPath + ": file lies outside module root " + moduleDir +
					" (generated/cgo intermediates are unsupported at this stage)"}
			}
			fileID, err := identity.NewFileID(moduleID, filepath.ToSlash(rel))
			if err != nil {
				return nil, err
			}
			out.files = append(out.files, &File{path: osPath, id: fileID, fset: fset, syntax: syntax})
		}
		sort.Slice(out.files, func(i, j int) bool { return out.files[i].id.Rel() < out.files[j].id.Rel() })
		ws.packages = append(ws.packages, out)
	}
	if len(ws.packages) == 0 {
		return nil, &LoadError{Dir: req.Dir, Reason: "no Go source packages matched " + strings.Join(patterns, " ")}
	}
	sort.Slice(ws.packages, func(i, j int) bool { return ws.packages[i].id.String() < ws.packages[j].id.String() })
	return ws, nil
}
