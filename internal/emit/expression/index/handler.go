package index

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	array, ok := arrayvalue.Resolve(
		context,
		context.TypesInfo().TypeOf(source.X),
	)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return array.EmitIndex(context, children, source)
}
