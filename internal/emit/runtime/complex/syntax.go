package complex

import "github.com/tsoniclang/gotots/internal/target/tsgo"

type builder struct {
	factory tsgo.Factory
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func (b builder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b builder) typeReference(name string) tsgo.TypeNode {
	return b.factory.TypeReferenceNode(b.id(name), nil)
}

func (b builder) parameter(
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		nil,
		nil,
		b.id(name),
		nil,
		targetType,
		nil,
	)
}

func (b builder) property(
	receiver tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return b.factory.PropertyAccessExpression(
		receiver,
		nil,
		b.id(name),
		tsgo.NodeFlagsNone,
	)
}

func (b builder) call(
	callee tsgo.Expression,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		callee,
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) binary(
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.factory.BinaryExpression(
		nil,
		left,
		nil,
		b.factory.BinaryOperatorToken(operator),
		right,
	)
}

func (b builder) prefix(
	operator tsgo.PrefixUnaryExpressionOperatorKind,
	operand tsgo.Expression,
) tsgo.PrefixUnaryExpression {
	return b.factory.PrefixUnaryExpression(operator, operand)
}

func (b builder) variable(
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(
					b.id(name),
					nil,
					targetType,
					value,
				),
			},
			flags,
		),
	)
}

func (b builder) assign(
	name string,
	value tsgo.Expression,
) tsgo.ExpressionStatement {
	return b.factory.ExpressionStatement(b.binary(
		b.id(name),
		tsgo.BinaryOperatorEqualsToken,
		value,
	))
}

func (b builder) mathCall(
	member string,
	value tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.property(b.id("Math"), member), value)
}

func (b builder) numberCall(
	member string,
	value tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.property(b.id("Number"), member), value)
}

func (b builder) objectIs(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.property(b.id("Object"), "is"), left, right)
}

func (b builder) zero() tsgo.NumericLiteral {
	return b.factory.NumericLiteral("0", tsgo.TokenFlagsNone)
}

func (b builder) one() tsgo.NumericLiteral {
	return b.factory.NumericLiteral("1", tsgo.TokenFlagsNone)
}

func (b builder) negativeZero() tsgo.PrefixUnaryExpression {
	return b.prefix(
		tsgo.PrefixUnaryExpressionOperatorKindMinusToken,
		b.zero(),
	)
}
