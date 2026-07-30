package channel

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	factory           tsgo.Factory
	channelName       string
	receiveName       string
	sendName          string
	caseName          string
	selectName        string
	selectReadyName   string
	selectAttemptName string
	panicName         string
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
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

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) integerInputType() tsgo.TypeNode {
	return b.unionType(
		b.numberType(),
		b.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBigIntKeyword,
		),
	)
}

func (b builder) booleanType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword)
}

func (b builder) objectType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword)
}

func (b builder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b builder) undefinedType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)
}

func (b builder) promiseType(result tsgo.TypeNode) tsgo.TypeNode {
	return b.typeReference("Promise", result)
}

func (b builder) arrayType(element tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.ArrayTypeNode(element)
}

func (b builder) tupleType(elements ...tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.TupleTypeNode(elements)
}

func (b builder) unionType(elements ...tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode(elements)
}

func (b builder) functionType(
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
) tsgo.TypeNode {
	return b.factory.FunctionTypeNode(nil, parameters, result)
}

func (b builder) typeParameter() tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(nil, b.id("T"), nil, nil, nil)
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

func (b builder) thisProperty(name string) tsgo.PropertyAccessExpression {
	return b.property(b.factory.ThisExpression(), name)
}

func (b builder) element(
	receiver tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return b.factory.ElementAccessExpression(
		receiver,
		nil,
		index,
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

func (b builder) toNumber(value tsgo.Expression) tsgo.CallExpression {
	return b.call(api.TargetIntrinsicNumber.Expression(b.factory), value)
}

func (b builder) methodCall(
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.call(b.property(receiver, name), arguments...)
}

func (b builder) staticCall(
	name string,
	member string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.methodCall(b.id(name), member, arguments...)
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

func (b builder) multiply(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorAsteriskToken, right)
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
	return b.binary(left, tsgo.BinaryOperatorBarBarToken, right)
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

func (b builder) increment(target tsgo.Expression) tsgo.Expression {
	return b.assign(target, b.add(target, b.number("1")))
}

func (b builder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) string(value string) tsgo.StringLiteral {
	return b.factory.StringLiteral(value, tsgo.TokenFlagsNone)
}

func (b builder) undefined() tsgo.Identifier {
	return b.id("undefined")
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

func (b builder) variableDeclaration(
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableDeclaration {
	return b.factory.VariableDeclaration(
		b.id(name),
		nil,
		targetType,
		value,
	)
}

func (b builder) expression(value tsgo.Expression) tsgo.ExpressionStatement {
	return b.factory.ExpressionStatement(value)
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

func (b builder) propertyDeclaration(
	modifiers []tsgo.ModifierLike,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.PropertyDeclaration {
	return b.factory.PropertyDeclaration(
		modifiers,
		b.id(name),
		nil,
		targetType,
		value,
	)
}

func (b builder) arrow(
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	body tsgo.ConciseBody,
) tsgo.ArrowFunction {
	return b.factory.ArrowFunction(
		nil,
		nil,
		parameters,
		result,
		b.factory.EqualsGreaterThanToken(),
		body,
	)
}

func (b builder) returnStatement(value tsgo.Expression) tsgo.ReturnStatement {
	return b.factory.ReturnStatement(value)
}

func (b builder) strictUndefined(value tsgo.Expression) tsgo.Expression {
	return b.binary(
		value,
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.undefined(),
	)
}

func (b builder) strictDefined(value tsgo.Expression) tsgo.Expression {
	return b.binary(
		value,
		tsgo.BinaryOperatorExclamationEqualsEqualsToken,
		b.undefined(),
	)
}

func (b builder) logicalNot(value tsgo.Expression) tsgo.Expression {
	return b.factory.PrefixUnaryExpression(
		tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
		value,
	)
}

func (b builder) newPromise(
	result tsgo.TypeNode,
	executor tsgo.Expression,
) tsgo.NewExpression {
	return b.factory.NewExpression(
		b.id("Promise"),
		[]tsgo.TypeNode{result},
		[]tsgo.Expression{executor},
	)
}

func (b builder) promiseResolve(value tsgo.Expression) tsgo.CallExpression {
	arguments := []tsgo.Expression{}
	if value != nil {
		arguments = append(arguments, value)
	}
	return b.staticCall("Promise", "resolve", arguments...)
}

func (b builder) arrayLiteral(values ...tsgo.Expression) tsgo.ArrayLiteralExpression {
	return b.factory.ArrayLiteralExpression(values, false)
}

func (b builder) arrayLength(
	value tsgo.Expression,
) tsgo.PropertyAccessExpression {
	return b.property(value, "length")
}

func (b builder) panic(value string) tsgo.CallExpression {
	return b.staticCall(
		b.panicName,
		"raiseRuntime",
		b.string(value),
	)
}

func (b builder) panicValue(value string) tsgo.CallExpression {
	return b.staticCall(
		b.panicName,
		"createRuntime",
		b.string(value),
	)
}

func (b builder) rethrow(value tsgo.Expression) tsgo.CallExpression {
	return b.staticCall(
		b.panicName,
		panicruntime.RethrowName,
		value,
	)
}

func (b builder) propertyFunction(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	body tsgo.ConciseBody,
) tsgo.PropertyAssignment {
	return b.factory.PropertyAssignment(
		nil,
		b.id(name),
		nil,
		b.functionType(parameters, result),
		b.arrow(parameters, result, body),
	)
}
