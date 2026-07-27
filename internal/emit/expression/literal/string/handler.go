package stringliteral

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.BasicLit,
) (api.ExpressionEmission, error) {
	typeAndValue, ok := context.TypesInfo().Types[source]
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
	if !basictype.SupportsString(targetType) ||
		!types.AssignableTo(typeAndValue.Type, targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.DirectExpression(
		context.Factory().StringLiteral(
			byteCodeUnits(constant.StringVal(typeAndValue.Value)),
			tsgo.TokenFlagsNone,
		),
	), nil
}

func byteCodeUnits(value string) string {
	bytes := []byte(value)
	codeUnits := make([]rune, len(bytes))
	for index, value := range bytes {
		codeUnits[index] = rune(value)
	}
	return string(codeUnits)
}
