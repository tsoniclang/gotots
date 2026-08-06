package unsafepointer

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type builder struct {
	factory           tsgo.Factory
	className         string
	codecName         string
	panicName         string
	pointerName       string
	pointerMemoryName string
	denseIndexName    string
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

func (b builder) undefined() tsgo.Expression {
	return b.factory.VoidExpression(b.number("0"))
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) bigintType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBigIntKeyword)
}

func (b builder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b builder) objectType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword)
}

func (b builder) byteArrayType() tsgo.TypeNode {
	return b.typeReference("Uint8Array")
}

func (b builder) unsafeType() tsgo.TypeNode {
	return b.typeReference(b.className)
}

func (b builder) optional(target tsgo.TypeNode) tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		target,
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}

func (b builder) typeReference(
	name string,
	arguments ...tsgo.TypeNode,
) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), arguments)
}

func (b builder) typeParameter(
	name string,
	constraint tsgo.TypeNode,
) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(
		nil,
		b.id(name),
		constraint,
		nil,
		nil,
	)
}

func (b builder) parameter(
	name string,
	target tsgo.TypeNode,
	modifiers []tsgo.ModifierLike,
) tsgo.ParameterDeclaration {
	return b.factory.ParameterDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		target,
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
	receiver tsgo.Expression,
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) typedCall(
	receiver tsgo.Expression,
	name string,
	types []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.property(receiver, name),
		nil,
		types,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func (b builder) directCall(
	name string,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		b.id(name),
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

func (b builder) assign(
	left tsgo.Expression,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return b.binary(left, tsgo.BinaryOperatorEqualsToken, right)
}

func (b builder) variable(
	flags tsgo.NodeFlags,
	name string,
	target tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return b.factory.VariableStatement(
		nil,
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
				b.id(name),
				nil,
				target,
				value,
			)},
			flags,
		),
	)
}

func (b builder) method(
	modifiers []tsgo.ModifierLike,
	name string,
	typeParameters []tsgo.TypeParameterDeclaration,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	var body tsgo.Block
	if statements != nil {
		body = b.factory.Block(statements, true)
	}
	return b.factory.MethodDeclaration(
		modifiers,
		nil,
		b.id(name),
		nil,
		typeParameters,
		parameters,
		result,
		body,
	)
}

func (b builder) panic(message string) tsgo.Statement {
	return b.factory.ExpressionStatement(panicruntime.Call(
		b.factory,
		b.panicName,
		b.string(message),
	))
}

func (b builder) notSafeInteger(value tsgo.Expression) tsgo.Expression {
	return b.factory.PrefixUnaryExpression(
		tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
		b.call(
			b.property(b.id("globalThis"), "Number"),
			"isSafeInteger",
			value,
		),
	)
}

func (b builder) readBytesType() tsgo.TypeNode {
	return b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
			b.parameter("length", b.numberType(), nil),
		},
		b.byteArrayType(),
	)
}

func (b builder) writeBytesType() tsgo.TypeNode {
	return b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("offset", b.numberType(), nil),
			b.parameter("bytes", b.byteArrayType(), nil),
		},
		b.voidType(),
	)
}

func (b builder) childrenType() tsgo.TypeNode {
	return b.typeReference("Map", b.numberType(), b.unsafeType())
}

func (b builder) codecType(storage tsgo.TypeNode) tsgo.TypeNode {
	return b.typeReference(b.codecName, storage)
}
