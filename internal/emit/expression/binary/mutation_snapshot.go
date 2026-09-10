package binary

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/expression/operands"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, children api.ChildEmitter, source *ast.BinaryExpr) (api.ExpressionEmission, error) {
	value, err := emitBinary(context, children, source)
	if err != nil || source.Op != token.EQL && source.Op != token.NEQ ||
		!context.IndirectlyMutable(source.X) && !context.IndirectlyMutable(source.Y) {
		return value, err
	}
	return operands.Snapshot(context, value, api.DirectType(context.Factory().KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)))
}
