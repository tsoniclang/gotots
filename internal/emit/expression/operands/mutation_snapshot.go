package operands

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BooleanMutationSnapshot(context api.Context, source ast.Expr, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	if !context.IndirectlyMutable(source) {
		return value, nil
	}
	return Snapshot(context, value, api.DirectType(context.Factory().KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)))
}

func Snapshot(context api.Context, value api.ExpressionEmission, valueType api.TypeEmission) (api.ExpressionEmission, error) {
	if valueType.Value() == nil {
		return api.ExpressionEmission{}, &api.InvariantError{Role: context.Role(), Reason: "value snapshot has no selected type"}
	}
	name, err := context.Names().Temporary(api.TemporaryLogicalResult)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(value.Before(), context.Factory().VariableStatement(nil,
		context.Factory().VariableDeclarationList([]tsgo.VariableDeclaration{
			context.Factory().VariableDeclaration(context.Factory().Identifier(name), nil,
				valueType.Value(), value.Value()),
		}, tsgo.NodeFlagsLet),
	))
	return api.NewExpressionEmission(before, context.Factory().Identifier(name), api.CombineRequests(value.Requests(), valueType.Requests()))
}
