package load

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/toolchain"
)

func (f File) Path() string           { return f.path }
func (f File) SourceIdentity() string { return f.sourceIdentity }
func (f File) SourceDigest() string   { return f.sourceDigest }
func (f File) Syntax() *ast.File      { return f.syntax }

func (p *Package) Path() string          { return p.path }
func (p *Package) Name() string          { return p.name }
func (p *Package) ModulePath() string    { return p.modulePath }
func (p *Package) ModuleVersion() string { return p.moduleVersion }
func (p *Package) SourceRoot() string    { return p.sourceRoot }
func (p *Package) Owner() PackageOwner   { return p.owner }
func (p *Package) OwnerKey() string      { return p.ownerKey }

func (p *Package) ToolchainKey() string {
	if p.kind != PackageStandardLibraryContract {
		return ""
	}
	return p.contractKey
}

func (p *Package) ExternalContractKey() string {
	if p.kind != PackageExternalContract {
		return ""
	}
	return p.contractKey
}

func (p *Package) Kind() PackageKind       { return p.kind }
func (p *Package) Files() []File           { return slices.Clone(p.files) }
func (p *Package) OtherFiles() []string    { return slices.Clone(p.otherFiles) }
func (p *Package) FileSet() *token.FileSet { return p.fileSet }
func (p *Package) Types() *types.Package   { return p.typesPackage }
func (p *Package) TypesInfo() *types.Info  { return p.typesInfo }
func (p *Package) TypesSizes() types.Sizes { return p.typesSizes }
func (p *Package) Program() *Program       { return p.program }

func (p *Package) SyntaxParent(source ast.Node) (ast.Node, bool) {
	if p == nil || source == nil {
		return nil, false
	}
	parent, ok := p.syntaxParents[source]
	return parent, ok
}

func (p *Package) FileForSyntax(syntax *ast.File) (File, bool) {
	for _, file := range p.files {
		if file.syntax == syntax {
			return file, true
		}
	}
	return File{}, false
}

func (p *Program) Roots() []*Package                              { return slices.Clone(p.roots) }
func (p *Program) Packages() []*Package                           { return slices.Clone(p.packages) }
func (p *Program) EnvironmentPackages() []*Package                { return slices.Clone(p.environmentPackages) }
func (p *Program) PackageByPath(path string) *Package             { return p.byPath[path] }
func (p *Program) PackageForTypes(source *types.Package) *Package { return p.byTypes[source] }
func (p *Program) EnvironmentForTypes(source *types.Package) *Package {
	return p.environmentByTypes[source]
}

func (p *Program) BuildProfile() BuildProfile {
	if p == nil {
		return BuildProfile{}
	}
	return p.buildProfile
}

func (p *Program) GoTool() toolchain.Go {
	if p == nil {
		return toolchain.Go{}
	}
	return p.goTool
}

func (p *Program) SourceDigest() string {
	if p == nil {
		return ""
	}
	return p.sourceDigest
}
