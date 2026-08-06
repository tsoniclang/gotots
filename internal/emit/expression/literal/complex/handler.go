package complex

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
	if source == nil || source.Kind != token.IMAG {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	facts, ok := context.TypesInfo().TypeAndValue(source)
	if !ok || facts.Value == nil || facts.Value.Kind() != constant.Complex {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetType := context.ExpectedType()
	if targetType == nil {
		targetType = facts.Type
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
		facts.Value,
	)
}
