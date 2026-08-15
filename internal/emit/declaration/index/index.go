package index

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
)

func FileObjects(
	sourcePackage *load.Package,
	sourceFile *ast.File,
) ([]types.Object, error) {
	if sourcePackage == nil || sourcePackage.Program() == nil || sourceFile == nil {
		return nil, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "file-root source is invalid",
		}
	}
	if _, ok := sourcePackage.FileForSyntax(sourceFile); !ok {
		return nil, &api.InvariantError{
			Role:   api.RoleFileDeclaration,
			Reason: "file-root syntax is not package-owned",
		}
	}
	var objects []types.Object
	for _, declaration := range sourceFile.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if IsPackageInitializer(declaration) {
				continue
			}
			object := sourcePackage.TypesInfo().Defs[declaration.Name]
			if object == nil {
				return nil, &api.InvariantError{
					Role:   api.RoleFileDeclaration,
					Reason: "function declaration has no object identity",
				}
			}
			objects = append(objects, object)
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						object := sourcePackage.TypesInfo().Defs[name]
						if object != nil && object.Name() != "_" {
							objects = append(objects, object)
						}
					}
				case *ast.TypeSpec:
					if object := sourcePackage.TypesInfo().Defs[spec.Name]; object != nil {
						objects = append(objects, object)
					}
				}
			}
		}
	}
	sort.Slice(objects, func(left, right int) bool {
		if objects[left].Pos() != objects[right].Pos() {
			return objects[left].Pos() < objects[right].Pos()
		}
		return objects[left].Name() < objects[right].Name()
	})
	return objects, nil
}

type Site struct {
	Object      types.Object
	Source      *load.Package
	SourceFile  load.File
	Declaration ast.Decl
	OutputPath  string
}

type DuplicateError struct {
	Object string
}

func (e *DuplicateError) Error() string {
	return fmt.Sprintf(
		"index declaration %q: object has multiple source declarations",
		e.Object,
	)
}

func Program(source *load.Program) (map[types.Object]Site, error) {
	sites := make(map[types.Object]Site)
	for _, sourcePackage := range source.Packages() {
		for _, sourceFile := range sourcePackage.Files() {
			outputPath, err := targetoutput.SourcePath(sourcePackage, sourceFile)
			if err != nil {
				return nil, err
			}
			for _, declaration := range sourceFile.Syntax().Decls {
				switch declaration := declaration.(type) {
				case *ast.FuncDecl:
					object, ok := sourcePackage.TypesInfo().
						Defs[declaration.Name].(*types.Func)
					if !ok {
						return nil, &api.InvariantError{
							Role:   api.RoleFileDeclaration,
							Reason: "function declaration has no go/types object",
						}
					}
					if object.Name() == "_" {
						continue
					}
					if err := addSite(
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

func IsPackageInitializer(declaration *ast.FuncDecl) bool {
	return declaration != nil &&
		declaration.Name != nil &&
		declaration.Name.Name == "init" &&
		declaration.Recv == nil
}

func indexGeneralDeclaration(
	sites map[types.Object]Site,
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
				if err := addSite(
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
							Role: api.RoleFileDeclaration,
							Reason: "blank package variable has no exact " +
								"go/types object",
						}
					}
					if err := addSite(
						sites,
						object,
						sourcePackage,
						sourceFile,
						declaration,
						outputPath,
					); err != nil {
						return err
					}
					continue
				}
				if !ok ||
					object.IsField() ||
					object.Parent() != sourcePackage.Types().Scope() {
					return &api.InvariantError{
						Role: api.RoleFileDeclaration,
						Reason: "variable declaration has no package-scope " +
							"go/types object",
					}
				}
				if err := addSite(
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
			object, ok := sourcePackage.TypesInfo().
				Defs[typeSpec.Name].(*types.TypeName)
			if !ok {
				return &api.InvariantError{
					Role:   api.RoleFileDeclaration,
					Reason: "type declaration has no go/types object",
				}
			}
			if object.Name() == "_" {
				continue
			}
			if err := addSite(
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

func addSite(
	sites map[types.Object]Site,
	object types.Object,
	sourcePackage *load.Package,
	sourceFile load.File,
	declaration ast.Decl,
	outputPath string,
) error {
	if _, duplicate := sites[object]; duplicate {
		return &DuplicateError{Object: object.Name()}
	}
	sites[object] = Site{
		Object:      object,
		Source:      sourcePackage,
		SourceFile:  sourceFile,
		Declaration: declaration,
		OutputPath:  outputPath,
	}
	return nil
}

func CompareSites(left Site, right Site) int {
	switch {
	case left.Source.Path() < right.Source.Path():
		return -1
	case left.Source.Path() > right.Source.Path():
		return 1
	case left.OutputPath < right.OutputPath:
		return -1
	case left.OutputPath > right.OutputPath:
		return 1
	case left.Object.Pos() < right.Object.Pos():
		return -1
	case left.Object.Pos() > right.Object.Pos():
		return 1
	case left.Object.Name() < right.Object.Name():
		return -1
	case left.Object.Name() > right.Object.Name():
		return 1
	default:
		return 0
	}
}
