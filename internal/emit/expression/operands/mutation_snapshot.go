package operands

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func MutationSnapshot(context api.Context, source ast.Expr, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	if !context.IndirectlyMutable(source) {
		return value, nil
	}
	return Snapshot(context, value)
}

func Snapshot(context api.Context, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryLogicalResult)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(value.Before(), context.Factory().VariableStatement(nil,
		context.Factory().VariableDeclarationList([]tsgo.VariableDeclaration{
			context.Factory().VariableDeclaration(context.Factory().Identifier(name), nil, nil, value.Value()),
		}, tsgo.NodeFlagsLet),
	))
	return api.NewExpressionEmission(before, context.Factory().Identifier(name), value.Requests())
}
