package emit

import (
	"go/ast"

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
	return &Emitter{
		source:  source,
		factory: tsgo.NewFactory(),
		names:   newNameOwner(),
	}
}

func (e *Emitter) EmitFile(sourceFile *ast.File, outputPath string) (tsgo.SourceFile, error) {
	if e == nil || e.source == nil {
		return nil, &Error{Reason: "source package is nil"}
	}
	if !e.ownsFile(sourceFile) {
		return nil, &Error{Reason: "syntax tree does not belong to the loaded package"}
	}

	placement := newPlacementOwner()
	context, err := api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		e.factory,
		e.names,
		placement,
	)
	if err != nil {
		return nil, err
	}
	declarations := make([]tsgo.Statement, 0, len(sourceFile.Decls))
	for _, declaration := range sourceFile.Decls {
		statement, err := e.declaration(context, declaration)
		if err != nil {
			return nil, err
		}
		declarations = append(declarations, statement)
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

func (e *Emitter) ownsFile(candidate *ast.File) bool {
	for _, sourceFile := range e.source.Syntax() {
		if sourceFile == candidate {
			return true
		}
	}
	return false
}
