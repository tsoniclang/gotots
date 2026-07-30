package emit

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
)

type declarationSite struct {
	object      types.Object
	source      *load.Package
	sourceFile  load.File
	declaration ast.Decl
	outputPath  string
}

func indexDeclarations(source *load.Program) (map[types.Object]declarationSite, error) {
	sites := make(map[types.Object]declarationSite)
	for _, sourcePackage := range source.Packages() {
		for _, sourceFile := range sourcePackage.Files() {
			outputPath, err := targetoutput.SourcePath(sourcePackage, sourceFile)
			if err != nil {
				return nil, err
			}
			for _, declaration := range sourceFile.Syntax().Decls {
				switch declaration := declaration.(type) {
				case *ast.FuncDecl:
					object, ok := sourcePackage.TypesInfo().Defs[declaration.Name].(*types.Func)
					if !ok {
						return nil, &api.InvariantError{
							Role:   api.RoleFileDeclaration,
							Reason: "function declaration has no go/types object",
						}
					}
					if object.Name() == "_" {
						continue
					}
					if err := addDeclarationSite(
						sites,
						object,
						sourcePackage,
						sourceFile,
						declaration,
						outputPath,
					); err != nil {
						return nil, err
					}
				case *ast.GenDecl:
					if err := indexGeneralDeclaration(
						sites,
						sourcePackage,
						sourceFile,
						declaration,
						outputPath,
					); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	return sites, nil
}

func isPackageInitDeclaration(declaration *ast.FuncDecl) bool {
	return declaration != nil &&
		declaration.Name != nil &&
		declaration.Name.Name == "init" &&
		declaration.Recv == nil
}

func indexGeneralDeclaration(
	sites map[types.Object]declarationSite,
	sourcePackage *load.Package,
	sourceFile load.File,
	declaration *ast.GenDecl,
	outputPath string,
) error {
	switch declaration.Tok {
	case token.CONST:
		for _, spec := range declaration.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				return &api.InvariantError{
					Role:   api.RoleFileDeclaration,
					Reason: "constant declaration has a non-value spec",
				}
			}
			for _, name := range valueSpec.Names {
				object, ok := sourcePackage.TypesInfo().Defs[name].(*types.Const)
				if !ok {
					return &api.InvariantError{
						Role:   api.RoleFileDeclaration,
						Reason: "constant declaration has no go/types object",
					}
				}
				if object.Name() == "_" {
					continue
				}
				if err := addDeclarationSite(
					sites,
					object,
					sourcePackage,
					sourceFile,
					declaration,
					outputPath,
				); err != nil {
					return err
				}
			}
		}
	case token.VAR:
		sourceOutputPath := outputPath
		statePath, err := targetoutput.PackageStatePath(sourcePackage)
		if err != nil {
			return err
		}
		for _, spec := range declaration.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				return &api.InvariantError{
					Role:   api.RoleFileDeclaration,
					Reason: "variable declaration has a non-value spec",
				}
			}
			for _, name := range valueSpec.Names {
				object, ok := sourcePackage.TypesInfo().Defs[name].(*types.Var)
				if name.Name == "_" {
					if !ok ||
						object.Name() != "_" ||
						object.Pkg() != sourcePackage.Types() ||
						object.Parent() != nil {
						return &api.InvariantError{
							Role:   api.RoleFileDeclaration,
							Reason: "blank package variable has no exact go/types object",
						}
					}
					if err := addDeclarationSite(
						sites,
						object,
						sourcePackage,
						sourceFile,
						declaration,
						sourceOutputPath,
					); err != nil {
						return err
					}
					continue
				}
				if !ok ||
					object.IsField() ||
					object.Parent() != sourcePackage.Types().Scope() {
					return &api.InvariantError{
						Role:   api.RoleFileDeclaration,
						Reason: "variable declaration has no package-scope go/types object",
					}
				}
				if err := addDeclarationSite(
					sites,
					object,
					sourcePackage,
					sourceFile,
					declaration,
					statePath,
				); err != nil {
					return err
				}
			}
		}
	case token.TYPE:
		for _, spec := range declaration.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				return &api.InvariantError{
					Role:   api.RoleFileDeclaration,
					Reason: "type declaration has a non-type spec",
				}
			}
			object, ok := sourcePackage.TypesInfo().Defs[typeSpec.Name].(*types.TypeName)
			if !ok {
				return &api.InvariantError{
					Role:   api.RoleFileDeclaration,
					Reason: "type declaration has no go/types object",
				}
			}
			if object.Name() == "_" {
				continue
			}
			if err := addDeclarationSite(
				sites,
				object,
				sourcePackage,
				sourceFile,
				declaration,
				outputPath,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func addDeclarationSite(
	sites map[types.Object]declarationSite,
	object types.Object,
	sourcePackage *load.Package,
	sourceFile load.File,
	declaration ast.Decl,
	outputPath string,
) error {
	if _, duplicate := sites[object]; duplicate {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object has multiple source declarations",
		}
	}
	sites[object] = declarationSite{
		object:      object,
		source:      sourcePackage,
		sourceFile:  sourceFile,
		declaration: declaration,
		outputPath:  outputPath,
	}
	return nil
}

func compareDeclarationSites(left declarationSite, right declarationSite) int {
	switch {
	case left.source.Path() < right.source.Path():
		return -1
	case left.source.Path() > right.source.Path():
		return 1
	case left.outputPath < right.outputPath:
		return -1
	case left.outputPath > right.outputPath:
		return 1
	case left.object.Pos() < right.object.Pos():
		return -1
	case left.object.Pos() > right.object.Pos():
		return 1
	case left.object.Name() < right.object.Name():
		return -1
	case left.object.Name() > right.object.Name():
		return 1
	default:
		return 0
	}
}
