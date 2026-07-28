package stringruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func variable(
	factory tsgo.Factory,
	flags tsgo.NodeFlags,
	name tsgo.Identifier,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(name, nil, nil, value),
			},
			flags,
		),
	)
}

func assignStatement(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.ExpressionStatement {
	return factory.ExpressionStatement(
		binary(factory, left, tsgo.BinaryOperatorEqualsToken, right),
	)
}

func callIdentifier(
	factory tsgo.Factory,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.Identifier(name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func fromCharCode(
	factory tsgo.Factory,
	values ...tsgo.Expression,
) tsgo.CallExpression {
	return methodCall(
		factory,
		factory.Identifier("String"),
		"fromCharCode",
		values,
	)
}

func charCodeAt(
	factory tsgo.Factory,
	value tsgo.Expression,
	index tsgo.Expression,
) tsgo.CallExpression {
	return methodCall(factory, value, "charCodeAt", []tsgo.Expression{index})
}

func continuationByte(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(
		factory,
		numeric(factory, "128"),
		tsgo.BinaryOperatorBarToken,
		binary(
			factory,
			value,
			tsgo.BinaryOperatorAmpersandToken,
			numeric(factory, "63"),
		),
	)
}

func invalidContinuation(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return or(
		factory,
		lessThan(factory, value, numeric(factory, "128")),
		greaterThan(factory, value, numeric(factory, "191")),
	)
}

func firstMask(
	factory tsgo.Factory,
	width tsgo.Expression,
) tsgo.Expression {
	return factory.ConditionalExpression(
		binary(
			factory,
			width,
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			numeric(factory, "2"),
		),
		factory.QuestionToken(),
		numeric(factory, "31"),
		factory.ColonToken(),
		factory.ConditionalExpression(
			binary(
				factory,
				width,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				numeric(factory, "3"),
			),
			factory.QuestionToken(),
			numeric(factory, "15"),
			factory.ColonToken(),
			numeric(factory, "7"),
		),
	)
}

func runeTuple(
	factory tsgo.Factory,
	value string,
	width string,
) tsgo.ArrayLiteralExpression {
	return factory.ArrayLiteralExpression(
		[]tsgo.Expression{numeric(factory, value), numeric(factory, width)},
		false,
	)
}

func notInteger(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.PrefixUnaryExpression {
	return factory.PrefixUnaryExpression(
		tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
		methodCall(
			factory,
			factory.Identifier("Number"),
			"isInteger",
			[]tsgo.Expression{value},
		),
	)
}

func and(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(factory, left, tsgo.BinaryOperatorAmpersandAmpersandToken, right)
}

func lessThanOrEqual(
	factory tsgo.Factory,
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return binary(factory, left, tsgo.BinaryOperatorLessThanEqualsToken, right)
}
