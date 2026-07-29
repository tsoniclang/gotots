package stringvalue

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func conversionNames(
	context api.Context,
) (string, string, string, error) {
	names := make([]string, 3)
	for index := range names {
		name, err := context.Names().Temporary(
			api.TemporaryConversionOperand,
		)
		if err != nil {
			return "", "", "", err
		}
		names[index] = name
	}
	return names[0], names[1], names[2], nil
}

func variable(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}

func property(
	context api.Context,
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		receiver,
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}

func callMember(
	context api.Context,
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		property(context, receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func binary(
	context api.Context,
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return context.Factory().BinaryExpression(
		nil,
		left,
		nil,
		context.Factory().BinaryOperatorToken(operator),
		right,
	)
}

func forLoop(
	context api.Context,
	index tsgo.Identifier,
	length tsgo.Expression,
	body []tsgo.Statement,
) tsgo.ForStatement {
	return context.Factory().ForStatement(
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					index,
					nil,
					nil,
					context.Factory().NumericLiteral(
						"0",
						tsgo.TokenFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsLet,
		),
		binary(
			context,
			index,
			tsgo.BinaryOperatorLessThanToken,
			length,
		),
		context.Factory().PostfixUnaryExpression(
			index,
			tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken,
		),
		context.Factory().Block(body, true),
	)
}
