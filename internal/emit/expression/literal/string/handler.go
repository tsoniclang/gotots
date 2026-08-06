package stringliteral

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
	source *ast.BasicLit,
) (api.ExpressionEmission, error) {
	typeAndValue, ok := context.TypesInfo().TypeAndValue(source)
	if source.Kind != token.STRING ||
		!ok ||
		typeAndValue.Value == nil ||
		typeAndValue.Value.Kind() != constant.String {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.ExpectedType()
	if targetType == nil {
		targetType = typeAndValue.Type
	}
	if !types.AssignableTo(typeAndValue.Type, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return constantvalue.EmitValue(context, source, targetType, typeAndValue.Value)
}
