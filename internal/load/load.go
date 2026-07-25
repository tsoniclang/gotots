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

type Package struct {
	path            string
	name            string
	compiledGoFiles []string
	syntax          []*ast.File
	fileSet         *token.FileSet
	typesPackage    *types.Package
	typesInfo       *types.Info
	typesSizes      types.Sizes
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

	return &Package{
		path:            selected.PkgPath,
		name:            selected.Name,
		compiledGoFiles: slices.Clone(selected.CompiledGoFiles),
		syntax:          slices.Clone(selected.Syntax),
		fileSet:         selected.Fset,
		typesPackage:    selected.Types,
		typesInfo:       selected.TypesInfo,
		typesSizes:      selected.TypesSizes,
	}, nil
}

func (p *Package) Path() string {
	return p.path
}

func (p *Package) Name() string {
	return p.name
}

func (p *Package) CompiledGoFiles() []string {
	return slices.Clone(p.compiledGoFiles)
}

func (p *Package) Syntax() []*ast.File {
	return slices.Clone(p.syntax)
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
