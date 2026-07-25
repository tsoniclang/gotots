package returnstatement

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ReturnStmt,
) (tsgo.ReturnStatement, error) {
	results := context.FunctionResults()
	if len(source.Results) != 1 || results == nil || results.Len() != 1 {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	result, err := children.Expression(
		context.
			WithRole(api.RoleReturnResult).
			WithExpectedType(results.At(0).Type()),
		source.Results[0],
	)
	if err != nil {
		return nil, err
	}
	return context.Factory().ReturnStatement(result), nil
}
