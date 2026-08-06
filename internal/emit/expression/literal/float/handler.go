package float

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
)

// Emit materializes a floating-point literal at its contextual float type
// through the single constant-value owner. The literal's value is the checker's
// canonical go/constant value; the source spelling is never re-evaluated.
func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	if !isFloatLiteralSyntax(source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	typeAndValue, ok := context.TypesInfo().TypeAndValue(source)
	if !ok || typeAndValue.Value == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if kind := typeAndValue.Value.Kind(); kind != constant.Float && kind != constant.Int {
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

func isFloatLiteralSyntax(source ast.Expr) bool {
	literal, ok := source.(*ast.BasicLit)
	return ok && literal.Kind == token.FLOAT
}
