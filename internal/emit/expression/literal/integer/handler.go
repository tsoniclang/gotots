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
	typeAndValue, ok := context.TypesInfo().TypeAndValue(source)
	if !ok || typeAndValue.Value == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	// Integer syntax may carry a float value when the checker converts an untyped
	// integer literal to a float target (`var x float64 = 8`). The single value
	// owner materializes it at the target representation; a non-numeric value is
	// never integer syntax.
	if kind := typeAndValue.Value.Kind(); kind != constant.Int && kind != constant.Float {
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
		// A rune literal is an int32 Unicode code point: integer syntax whose
		// checker value is a constant.Int materialized at its integer target.
		return source.Kind == token.INT || source.Kind == token.CHAR
	case *ast.UnaryExpr:
		literal, ok := source.X.(*ast.BasicLit)
		return source.Op == token.SUB && ok && literal.Kind == token.INT
	default:
		return false
	}
}
