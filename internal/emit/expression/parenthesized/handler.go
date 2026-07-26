package parenthesized

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ParenExpr,
) (api.ExpressionEmission, error) {
	if source.X == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source)
	childType := context.TypesInfo().TypeOf(source.X)
	if sourceType == nil ||
		childType == nil ||
		!types.Identical(sourceType, childType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	child, err := children.Expression(
		context.WithRole(api.RoleParenthesizedValue),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		child.Before(),
		context.Factory().ParenthesizedExpression(child.Value()),
		child.Requests(),
	)
}
