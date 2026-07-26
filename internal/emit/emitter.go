package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Emitter struct {
	source  *load.Package
	factory tsgo.Factory
	names   *nameOwner
}

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return "emit Go file: " + e.Reason
}

func New(source *load.Package) *Emitter {
	var typesInfo *types.Info
	var packageScope *types.Scope
	if source != nil {
		typesInfo = source.TypesInfo()
		packageScope = source.Types().Scope()
	}
	return &Emitter{
		source:  source,
		factory: tsgo.NewFactory(),
		names:   newNameOwner(packageScope, typesInfo),
	}
}

func (e *Emitter) EmitFile(sourceFile *ast.File, outputPath string) (tsgo.SourceFile, error) {
	if e == nil || e.source == nil {
		return nil, &Error{Reason: "source package is nil"}
	}
	_, ok := e.source.FileForSyntax(sourceFile)
	if !ok {
		return nil, &Error{Reason: "syntax tree does not belong to the loaded package"}
	}
	if err := e.reservePackageDeclarations(); err != nil {
		return nil, err
	}

	placement := newPlacementOwner()
	names := e.names.ForFile(sourceFile, e.source.Types().Scope(), e.factory)
	context, err := api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		e.factory,
		names,
	)
	if err != nil {
		return nil, err
	}
	declarations := make([]tsgo.Statement, 0, len(sourceFile.Decls))
	for _, declaration := range sourceFile.Decls {
		result, err := e.declaration(context, declaration)
		if err != nil {
			return nil, err
		}
		if err := placement.Apply(result.Requests()); err != nil {
			return nil, err
		}
		declarations = append(declarations, result.Declarations()...)
	}
	statements := append(placement.Statements(e.factory), declarations...)

	path, err := tsgo.NewPath(outputPath)
	if err != nil {
		return nil, err
	}
	return e.factory.SourceFile(
		statements,
		e.factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:   path,
			Path:       path,
			ScriptKind: tsgo.ScriptKindTS,
		},
	), nil
}

func (e *Emitter) reservePackageDeclarations() error {
	for _, sourceFile := range e.source.Files() {
		modulePath, err := targetModulePath(sourceFile.Path())
		if err != nil {
			return err
		}
		for _, declaration := range sourceFile.Syntax().Decls {
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				object, ok := e.source.TypesInfo().Defs[declaration.Name].(*types.Func)
				if !ok {
					return &api.InvariantError{
						Role:   api.RoleFileDeclaration,
						Reason: "function declaration has no go/types object",
					}
				}
				if _, err := e.names.Reserve(
					object,
					sourceFile.Syntax(),
					modulePath,
				); err != nil {
					return err
				}
			case *ast.GenDecl:
				if declaration.Tok != token.CONST {
					continue
				}
				for _, spec := range declaration.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						return &api.InvariantError{
							Role:   api.RoleFileDeclaration,
							Reason: "constant declaration has a non-value spec",
						}
					}
					for _, name := range valueSpec.Names {
						object, ok := e.source.TypesInfo().Defs[name].(*types.Const)
						if !ok {
							return &api.InvariantError{
								Role:   api.RoleFileDeclaration,
								Reason: "constant declaration has no go/types object",
							}
						}
						if _, err := e.names.Reserve(
							object,
							sourceFile.Syntax(),
							modulePath,
						); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func targetModulePath(sourcePath string) (string, error) {
	extension := filepath.Ext(sourcePath)
	baseName := strings.TrimSuffix(filepath.Base(sourcePath), extension)
	if extension != ".go" || baseName == "" {
		return "", &api.PlacementError{
			ModulePath: sourcePath,
			Reason:     "source file does not have a target module name",
		}
	}
	return "./" + baseName + ".js", nil
}
