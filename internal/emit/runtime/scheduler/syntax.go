package scheduler

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	factory       tsgo.Factory
	schedulerName string
	panicName     string
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) string(value string) tsgo.StringLiteral {
	return b.factory.StringLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
}

func (b builder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindBooleanKeyword,
	)
}

func (b builder) objectType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindObjectKeyword,
	)
}

func (b builder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
}

func (b builder) undefinedType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
	)
}

func (b builder) typeT() tsgo.TypeNode {
	return b.typeReference("T")
}

func (b builder) typeReference(
	name string,
	arguments ...tsgo.TypeNode,
) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), arguments)
}

func (b builder) promiseType(result tsgo.TypeNode) tsgo.TypeNode {
	return b.typeReference("Promise", result)
}

func (b builder) functionType(
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
) tsgo.TypeNode {
	return b.factory.FunctionTypeNode(nil, parameters, result)
}

func (b builder) failureType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.objectType(),
		b.undefinedType(),
	})
}

func (b builder) finishType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.failureType()),
		},
		b.voidType(),
	)
}

func (b builder) finishStorageType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.finishType(),
		b.undefinedType(),
	})
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

func (b builder) typeParameter() tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(
		nil,
		b.id("T"),
		nil,
		nil,
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

func (b builder) schedulerProperty(
	name string,
) tsgo.PropertyAccessExpression {
	return b.property(b.id(b.schedulerName), name)
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

func (b builder) methodCall(
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.property(receiver, name), arguments...)
}

func (b builder) schedulerCall(
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.methodCall(b.id(b.schedulerName), name, arguments...)
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

func (b builder) assign(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorEqualsToken, right)
}

func (b builder) add(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorPlusToken, right)
}

func (b builder) subtract(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorMinusToken, right)
}

func (b builder) logicalAnd(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(
		left,
		tsgo.BinaryOperatorAmpersandAmpersandToken,
		right,
	)
}

func (b builder) logicalOr(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(
		left,
		tsgo.BinaryOperatorBarBarToken,
		right,
	)
}

func (b builder) strictEqual(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(
		left,
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		right,
	)
}

func (b builder) strictDefined(
	value tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(
		value,
		tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		b.id("undefined"),
	)
}

func (b builder) logicalNot(value tsgo.Expression) tsgo.Expression {
	return b.factory.PrefixUnaryExpression(
		tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
		value,
	)
}

func (b builder) expression(
	value tsgo.Expression,
) tsgo.ExpressionStatement {
	return b.factory.ExpressionStatement(value)
}

func (b builder) returnStatement(
	value tsgo.Expression,
) tsgo.ReturnStatement {
	return b.factory.ReturnStatement(value)
}

func (b builder) propertyDeclaration(
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		b.id(name),
		nil,
		targetType,
		value,
	)
}

func (b builder) method(
	modifiers []tsgo.ModifierLike,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		nil,
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}

func (b builder) arrow(
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.ArrowFunction {
	return b.factory.ArrowFunction(
		nil,
		nil,
		parameters,
		result,
		b.factory.EqualsGreaterThanToken(),
		b.factory.Block(statements, true),
	)
}

func (b builder) constStatement(
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
				b.id(name),
				nil,
				targetType,
				value,
			)},
			tsgo.NodeFlagsConst,
		),
	)
}

func (b builder) promiseResolve() tsgo.CallExpression {
	return b.methodCall(b.id("Promise"), "resolve")
}

func (b builder) panicValue(message string) tsgo.CallExpression {
	return b.methodCall(
		b.id(b.panicName),
		"createRuntime",
		b.string(message),
	)
}

func (b builder) rethrow(value tsgo.Expression) tsgo.CallExpression {
	return b.methodCall(
		b.id(b.panicName),
		panicruntime.RethrowName,
		value,
	)
}
