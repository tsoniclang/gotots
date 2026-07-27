package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, bool, error) {
	identifier, ok := source.Fun.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	builtin, ok := context.TypesInfo().Uses[identifier].(*types.Builtin)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	if builtin.Name() != "len" && builtin.Name() != "cap" {
		return api.ExpressionEmission{}, false, nil
	}
	if len(source.Args) != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	array, represented := arrayvalue.Resolve(
		context,
		context.TypesInfo().TypeOf(source.Args[0]),
	)
	if !represented {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	resultType := context.TypesInfo().TypeOf(source)
	if resultType == nil ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := array.EmitLength(context, children, source)
	return target, true, err
}
