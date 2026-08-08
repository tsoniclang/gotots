package rawpointer

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BindNullable(
	context api.Context,
	identity api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	value := identity.Value()
	before := identity.Before()
	if value.Kind() != tsgo.SyntaxKindIdentifier {
		name, err := context.Names().Temporary(api.TemporaryConversionOperand)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				)},
				tsgo.NodeFlagsConst,
			),
		))
		value = context.Factory().Identifier(name)
	}
	bound, err := Operation(
		context,
		tsoniccore.SymbolBindRawPointer,
		api.DirectExpression(value),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				value,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				context.Factory().Identifier("undefined"),
			),
			context.Factory().QuestionToken(),
			context.Factory().Identifier("undefined"),
			context.Factory().ColonToken(),
			bound.Value(),
		),
		api.CombineRequests(identity.Requests(), bound.Requests()),
	)
}
