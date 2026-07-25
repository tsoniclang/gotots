package emit

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	functiondeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/function"
)

func (e *Emitter) declaration(
	context api.Context,
	source ast.Decl,
) (api.DeclarationEmission, error) {
	switch source := source.(type) {
	case *ast.FuncDecl:
		return functiondeclaration.Emit(context, e, source)
	default:
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
}
