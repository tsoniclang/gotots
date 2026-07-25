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
	path         string
	name         string
	files        []File
	fileSet      *token.FileSet
	typesPackage *types.Package
	typesInfo    *types.Info
	typesSizes   types.Sizes
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
	if ctx == nil {
		return nil, &Error{Pattern: request.Pattern, Reason: "context is nil"}
	}
	if request.Directory == "" {
		return nil, &Error{Pattern: request.Pattern, Reason: "directory is empty"}
	}
	if request.Pattern == "" {
		return nil, &Error{Reason: "pattern is empty"}
	}

	loaded, err := packages.Load(&packages.Config{
		Context: ctx,
		Dir:     request.Directory,
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

	selected := loaded[0]
	if selected.Fset == nil || selected.Types == nil || selected.TypesInfo == nil || selected.TypesSizes == nil {
		return nil, &Error{Pattern: request.Pattern, Reason: "selected package lacks syntax or type evidence"}
	}
	if len(selected.Syntax) != len(selected.CompiledGoFiles) {
		return nil, &Error{
			Pattern: request.Pattern,
			Reason: fmt.Sprintf(
				"syntax/file mismatch: %d syntax trees for %d compiled files",
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
	return &Package{
		path:         selected.PkgPath,
		name:         selected.Name,
		files:        files,
		fileSet:      selected.Fset,
		typesPackage: selected.Types,
		typesInfo:    selected.TypesInfo,
		typesSizes:   selected.TypesSizes,
	}, nil
}

func (p *Package) Path() string {
	return p.path
}

func (p *Package) Name() string {
	return p.name
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
