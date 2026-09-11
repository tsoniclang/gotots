package stringvalue

import "github.com/tsoniclang/gotots/internal/target/tsgo"

type builder struct {
	factory tsgo.Factory
}

func (b builder) id(name string) tsgo.Identifier { return b.factory.Identifier(name) }
func (b builder) number(value string) tsgo.Expression {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}
func (b builder) text(value string) tsgo.Expression {
	return b.factory.StringLiteral(value, tsgo.TokenFlagsNone)
}
func (b builder) reference(name string, arguments ...tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.TypeReferenceNode(b.id(name), arguments)
}
func (b builder) textType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword)
}
func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}
func (b builder) integerType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{b.numberType(), b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword)})
}
func (b builder) undefined() tsgo.Expression { return b.factory.VoidExpression(b.number("0")) }
func (b builder) optional(value tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{value, b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)})
}
func (b builder) member(receiver tsgo.Expression, name string) tsgo.Expression {
	return b.factory.PropertyAccessExpression(receiver, nil, b.id(name), tsgo.NodeFlagsNone)
}
func (b builder) own(name string) tsgo.Expression { return b.member(b.factory.ThisExpression(), name) }
func (b builder) call(receiver tsgo.Expression, arguments ...tsgo.Expression) tsgo.Expression {
	return b.factory.CallExpression(receiver, nil, nil, arguments, tsgo.NodeFlagsNone)
}
func (b builder) binary(left tsgo.Expression, operator tsgo.BinaryOperator, right tsgo.Expression) tsgo.Expression {
	return b.factory.BinaryExpression(nil, left, nil, b.factory.BinaryOperatorToken(operator), right)
}
func (b builder) variable(name string, value tsgo.Expression) tsgo.Statement {
	return b.factory.VariableStatement(nil, b.factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		b.factory.VariableDeclaration(b.id(name), nil, nil, value),
	}, tsgo.NodeFlagsConst))
}
func (b builder) parameter(name string, value tsgo.TypeNode) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(nil, nil, b.id(name), nil, value, nil)
}
func (b builder) fieldParameter(name string, value tsgo.TypeNode) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration([]tsgo.ModifierLike{b.factory.PrivateKeyword(), b.factory.ReadonlyKeyword()}, nil, b.id(name), nil, value, nil)
}
func (b builder) method(name string, parameters []tsgo.ParameterDeclaration, result tsgo.TypeNode, statements ...tsgo.Statement) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(nil, nil, b.id(name), nil, nil, parameters, result, b.factory.Block(statements, true))
}
func (b builder) staticMethod(name string, parameters []tsgo.ParameterDeclaration, result tsgo.TypeNode, statements ...tsgo.Statement) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration([]tsgo.ModifierLike{b.factory.StaticKeyword()}, nil, b.id(name), nil, nil, parameters, result, b.factory.Block(statements, true))
}
func (b builder) result(value tsgo.Expression) tsgo.Statement {
	return b.factory.ReturnStatement(value)
}
func (b builder) invoke(value tsgo.Expression) tsgo.Statement {
	return b.factory.ExpressionStatement(value)
}
func (b builder) fail(message string) tsgo.Statement {
	return b.invoke(b.call(b.member(b.id("GoPanic"), "raiseRuntime"), b.text(message)))
}

func (b builder) exactArithmetic(left tsgo.Expression, operator tsgo.BinaryOperator, right tsgo.Expression) tsgo.Expression {
	number := func(value tsgo.Expression) tsgo.Expression {
		return b.binary(b.factory.TypeOfExpression(value), tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.text("number"))
	}
	return b.factory.ConditionalExpression(b.binary(number(left), tsgo.BinaryOperatorAmpersandAmpersandToken, number(right)),
		b.factory.QuestionToken(), b.binary(left, operator, right), b.factory.ColonToken(),
		b.binary(b.call(b.id("BigInt"), left), operator, b.call(b.id("BigInt"), right)))
}
