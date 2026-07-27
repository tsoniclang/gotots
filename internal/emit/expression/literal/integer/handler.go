package integer

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
)

func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	if !isIntegerLiteralSyntax(source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	typeAndValue, ok := context.TypesInfo().Types[source]
	if !ok || typeAndValue.Value == nil || typeAndValue.Value.Kind() != constant.Int {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.ExpectedType()
	if targetType == nil {
		targetType = typeAndValue.Type
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil || !types.AssignableTo(sourceType, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return constantvalue.EmitValue(
		context,
		source,
		targetType,
		typeAndValue.Value,
	)
}

func isIntegerLiteralSyntax(source ast.Expr) bool {
	switch source := source.(type) {
	case *ast.BasicLit:
		return source.Kind == token.INT
	case *ast.UnaryExpr:
		literal, ok := source.X.(*ast.BasicLit)
		return source.Op == token.SUB && ok && literal.Kind == token.INT
	default:
		return false
	}
}
