package load

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Request struct {
	Directory string
	Pattern   string
}

type File struct {
	path   string
	syntax *ast.File
}

func (f File) Path() string {
	return f.path
}

func (f File) Syntax() *ast.File {
	return f.syntax
}

type Package struct {
	path          string
	name          string
	modulePath    string
	moduleVersion string
	files         []File
	fileSet       *token.FileSet
	typesPackage  *types.Package
	typesInfo     *types.Info
	typesSizes    types.Sizes
	program       *Program
}

type Program struct {
	roots    []*Package
	packages []*Package
	byPath   map[string]*Package
	byTypes  map[*types.Package]*Package
}

type Error struct {
	Pattern string
	Reason  string
}

func (e *Error) Error() string {
	if e.Pattern == "" {
		return "load Go package: " + e.Reason
	}
	return fmt.Sprintf("load Go package %q: %s", e.Pattern, e.Reason)
}

func One(ctx context.Context, request Request) (*Package, error) {
	program, err := Load(ctx, request)
	if err != nil {
		return nil, err
	}
	return program.roots[0], nil
}

func Load(ctx context.Context, request Request) (*Program, error) {
	if ctx == nil {
		return nil, &Error{Pattern: request.Pattern, Reason: "context is nil"}
	}
	if request.Directory == "" {
		return nil, &Error{Pattern: request.Pattern, Reason: "directory is empty"}
	}
	if request.Pattern == "" {
		return nil, &Error{Reason: "pattern is empty"}
	}

	fileSet := token.NewFileSet()
	loaded, err := packages.Load(&packages.Config{
		Context: ctx,
		Dir:     request.Directory,
		Fset:    fileSet,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes |
			packages.NeedModule,
	}, request.Pattern)
	if err != nil {
		return nil, &Error{Pattern: request.Pattern, Reason: err.Error()}
	}
	if len(loaded) != 1 {
		return nil, &Error{
			Pattern: request.Pattern,
			Reason:  fmt.Sprintf("selected %d packages, want exactly 1", len(loaded)),
		}
	}
	if problems := packageProblems(loaded); len(problems) != 0 {
		return nil, &Error{Pattern: request.Pattern, Reason: strings.Join(problems, "; ")}
	}

	selectedSet := make(map[*packages.Package]struct{}, len(loaded))
	for _, root := range loaded {
		selectedSet[root] = struct{}{}
	}
	var sourcePackages []*packages.Package
	packages.Visit(loaded, nil, func(current *packages.Package) {
		_, selected := selectedSet[current]
		if current.Module != nil || selected {
			sourcePackages = append(sourcePackages, current)
		}
	})
	sort.Slice(sourcePackages, func(left, right int) bool {
		return sourcePackages[left].PkgPath < sourcePackages[right].PkgPath
	})

	program := &Program{
		packages: make([]*Package, 0, len(sourcePackages)),
		byPath:   make(map[string]*Package, len(sourcePackages)),
		byTypes:  make(map[*types.Package]*Package, len(sourcePackages)),
	}
	byLoaded := make(map[*packages.Package]*Package, len(sourcePackages))
	for _, current := range sourcePackages {
		sourcePackage, err := wrapPackage(request.Pattern, current, fileSet)
		if err != nil {
			return nil, err
		}
		if _, duplicate := program.byPath[sourcePackage.path]; duplicate {
			return nil, &Error{
				Pattern: request.Pattern,
				Reason:  "duplicate package path " + sourcePackage.path,
			}
		}
		program.packages = append(program.packages, sourcePackage)
		program.byPath[sourcePackage.path] = sourcePackage
		program.byTypes[sourcePackage.typesPackage] = sourcePackage
		sourcePackage.program = program
		byLoaded[current] = sourcePackage
	}
	program.roots = make([]*Package, 0, len(loaded))
	for _, root := range loaded {
		sourcePackage := byLoaded[root]
		if sourcePackage == nil {
			return nil, &Error{
				Pattern: request.Pattern,
				Reason:  "selected package is absent from the source universe",
			}
		}
		program.roots = append(program.roots, sourcePackage)
	}
	return program, nil
}

func wrapPackage(
	pattern string,
	selected *packages.Package,
	fileSet *token.FileSet,
) (*Package, error) {
	if selected.Fset != fileSet || selected.Types == nil ||
		selected.TypesInfo == nil || selected.TypesSizes == nil {
		return nil, &Error{
			Pattern: pattern,
			Reason:  "source-available package lacks coherent syntax or type evidence",
		}
	}
	if len(selected.Syntax) != len(selected.CompiledGoFiles) {
		return nil, &Error{
			Pattern: pattern,
			Reason: fmt.Sprintf(
				"syntax/file mismatch in %s: %d syntax trees for %d compiled files",
				selected.PkgPath,
				len(selected.Syntax),
				len(selected.CompiledGoFiles),
			),
		}
	}
	files := make([]File, len(selected.Syntax))
	for index := range selected.Syntax {
		files[index] = File{
			path:   selected.CompiledGoFiles[index],
			syntax: selected.Syntax[index],
		}
	}
	var modulePath string
	var moduleVersion string
	if selected.Module != nil {
		modulePath = selected.Module.Path
		moduleVersion = selected.Module.Version
	}
	return &Package{
		path:          selected.PkgPath,
		name:          selected.Name,
		modulePath:    modulePath,
		moduleVersion: moduleVersion,
		files:         files,
		fileSet:       selected.Fset,
		typesPackage:  selected.Types,
		typesInfo:     selected.TypesInfo,
		typesSizes:    selected.TypesSizes,
	}, nil
}

func (p *Package) Path() string {
	return p.path
}

func (p *Package) Name() string {
	return p.name
}

func (p *Package) ModulePath() string {
	return p.modulePath
}

func (p *Package) ModuleVersion() string {
	return p.moduleVersion
}

func (p *Package) Files() []File {
	return slices.Clone(p.files)
}

func (p *Package) FileForSyntax(syntax *ast.File) (File, bool) {
	for _, file := range p.files {
		if file.syntax == syntax {
			return file, true
		}
	}
	return File{}, false
}

func (p *Package) FileSet() *token.FileSet {
	return p.fileSet
}

func (p *Package) Types() *types.Package {
	return p.typesPackage
}

func (p *Package) TypesInfo() *types.Info {
	return p.typesInfo
}

func (p *Package) TypesSizes() types.Sizes {
	return p.typesSizes
}

func (p *Package) Program() *Program {
	return p.program
}

func (p *Program) Roots() []*Package {
	return slices.Clone(p.roots)
}

func (p *Program) Packages() []*Package {
	return slices.Clone(p.packages)
}

func (p *Program) PackageByPath(path string) *Package {
	return p.byPath[path]
}

func (p *Program) PackageForTypes(source *types.Package) *Package {
	return p.byTypes[source]
}

func packageProblems(roots []*packages.Package) []string {
	var problems []string
	packages.Visit(roots, nil, func(current *packages.Package) {
		for _, problem := range current.Errors {
			problems = append(problems, current.PkgPath+": "+problem.Msg)
		}
	})
	sort.Strings(problems)
	return problems
}
