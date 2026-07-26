package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	functiondeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/function"
	packageconstant "github.com/tsoniclang/gotots/internal/emit/declaration/packageconstant"
)

func (e *emitter) declarationObject(
	context api.Context,
	source ast.Decl,
	object types.Object,
) (api.DeclarationEmission, error) {
	switch source := source.(type) {
	case *ast.FuncDecl:
		function, ok := object.(*types.Func)
		if !ok || context.TypesInfo().Defs[source.Name] != function {
			return api.DeclarationEmission{},
				&api.InvariantError{
					Role:   context.Role(),
					Reason: "scheduled function does not own its declaration",
				}
		}
		return functiondeclaration.Emit(context, e, source)
	case *ast.GenDecl:
		constant, ok := object.(*types.Const)
		if !ok {
			return api.DeclarationEmission{},
				api.Unsupported(context, api.CategoryDeclaration, source)
		}
		return packageconstant.EmitObject(context, e, source, constant)
	default:
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, source)
	}
}
