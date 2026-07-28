package stringvalue

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// EmitConstant emits a string constant from its checker value as a
// byte-preserving TypeScript string literal, keyed by the target type. It reads
// the go/constant value, never the source spelling.
func EmitConstant(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	value constant.Value,
) (api.ExpressionEmission, error) {
	if value == nil ||
		value.Kind() != constant.String ||
		!basictype.SupportsString(targetType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return api.DirectExpression(
		context.Factory().StringLiteral(
			byteCodeUnits(constant.StringVal(value)),
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
