package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (tsgo.Expression, error) {
	if source.Op != token.ADD || !isInt(context.TypesInfo().TypeOf(source)) {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	left, err := children.Expression(context.WithRole(api.RoleBinaryLeft), source.X)
	if err != nil {
		return nil, err
	}
	right, err := children.Expression(context.WithRole(api.RoleBinaryRight), source.Y)
	if err != nil {
		return nil, err
	}
	return context.Factory().BinaryExpression(
		nil,
		left,
		nil,
		context.Factory().PlusToken(),
		right,
	), nil
}

func isInt(value types.Type) bool {
	if value == nil {
		return false
	}
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && basic.Kind() == types.Int
}
