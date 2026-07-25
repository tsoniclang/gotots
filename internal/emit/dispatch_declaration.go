package emit

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	functiondeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/function"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *Emitter) declaration(context api.Context, source ast.Decl) (tsgo.Statement, error) {
	switch source := source.(type) {
	case *ast.FuncDecl:
		return functiondeclaration.Emit(context, e, source)
	default:
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}
}
