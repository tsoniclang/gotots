package unsafecodec

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	ReadName   = "read"
	WriteName  = "write"
	DecodeName = "decode"
	EncodeName = "encode"
	BindName   = "$bind"
	BoundName  = "$bound"
)

type builder struct {
	factory   tsgo.Factory
	className string
	panicName string
}

func Build(
	factory tsgo.Factory,
	className string,
	panicName string,
) tsgo.ClassDeclaration {
	target := builder{
		factory:   factory,
		className: className,
		panicName: panicName,
	}
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		[]tsgo.TypeParameterDeclaration{target.typeParameter("S")},
		nil,
		[]tsgo.ClassElement{
			target.bindingsProperty(),
			target.constructor(),
			target.read(),
			target.write(),
			target.decode(),
			target.encode(),
			target.bind(),
			target.bound(),
		},
	)
}

func (b builder) constructor() tsgo.ConstructorDeclaration {
	storage := b.typeReference("S")
	readType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("bytes", b.byteArrayType(), nil),
			b.parameter("offset", b.numberType(), nil),
		},
		storage,
	)
	writeType := b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", storage, nil),
			b.parameter("bytes", b.byteArrayType(), nil),
			b.parameter("offset", b.numberType(), nil),
		},
		b.voidType(),
	)
	return b.factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"size",
				b.numberType(),
				[]tsgo.ModifierLike{
					b.factory.PublicKeyword(),
					b.factory.ReadonlyKeyword(),
				},
			),
			b.parameter(
				"readValue",
				readType,
				[]tsgo.ModifierLike{
					b.factory.PrivateKeyword(),
					b.factory.ReadonlyKeyword(),
				},
			),
			b.parameter(
				"writeValue",
				writeType,
				[]tsgo.ModifierLike{
					b.factory.PrivateKeyword(),
					b.factory.ReadonlyKeyword(),
				},
			),
		},
		nil,
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.binary(
					b.notSafeInteger(b.id("size")),
					tsgo.BinaryOperatorBarBarToken,
					b.binary(
						b.id("size"),
						tsgo.BinaryOperatorLessThanToken,
						b.number("0"),
					),
				),
				b.panic("unsafe codec size is invalid"),
				nil,
			),
		}, true),
	)
}

func (b builder) read() tsgo.MethodDeclaration {
	return b.method(
		ReadName,
		[]tsgo.ParameterDeclaration{
			b.parameter("bytes", b.byteArrayType(), nil),
			b.parameter("offset", b.numberType(), nil),
		},
		b.typeReference("S"),
		b.boundsCheck(),
		b.factory.ReturnStatement(b.call(
			b.factory.ThisExpression(),
			"readValue",
			b.id("bytes"),
			b.id("offset"),
		)),
	)
}

func (b builder) write() tsgo.MethodDeclaration {
	return b.method(
		WriteName,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeReference("S"), nil),
			b.parameter("bytes", b.byteArrayType(), nil),
			b.parameter("offset", b.numberType(), nil),
		},
		b.voidType(),
		b.boundsCheck(),
		b.factory.ExpressionStatement(b.call(
			b.factory.ThisExpression(),
			"writeValue",
			b.id("value"),
			b.id("bytes"),
			b.id("offset"),
		)),
	)
}

func (b builder) decode() tsgo.MethodDeclaration {
	return b.method(
		DecodeName,
		[]tsgo.ParameterDeclaration{
			b.parameter("bytes", b.byteArrayType(), nil),
		},
		b.typeReference("S"),
		b.factory.ReturnStatement(b.call(
			b.factory.ThisExpression(),
			ReadName,
			b.id("bytes"),
			b.number("0"),
		)),
	)
}

func (b builder) encode() tsgo.MethodDeclaration {
	return b.method(
		EncodeName,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeReference("S"), nil),
		},
		b.byteArrayType(),
		b.variable(
			"bytes",
			b.byteArrayType(),
			b.factory.NewExpression(
				b.id("Uint8Array"),
				nil,
				[]tsgo.Expression{b.property(b.factory.ThisExpression(), "size")},
			),
		),
		b.factory.ExpressionStatement(b.call(
			b.factory.ThisExpression(),
			WriteName,
			b.id("value"),
			b.id("bytes"),
			b.number("0"),
		)),
		b.factory.ReturnStatement(b.id("bytes")),
	)
}

func (b builder) boundsCheck() tsgo.Statement {
	offset := b.id("offset")
	return b.factory.IfStatement(
		b.binary(
			b.notSafeInteger(offset),
			tsgo.BinaryOperatorBarBarToken,
			b.binary(
				b.binary(
					offset,
					tsgo.BinaryOperatorLessThanToken,
					b.number("0"),
				),
				tsgo.BinaryOperatorBarBarToken,
				b.binary(
					b.binary(
						offset,
						tsgo.BinaryOperatorPlusToken,
						b.property(b.factory.ThisExpression(), "size"),
					),
					tsgo.BinaryOperatorGreaterThanToken,
					b.property(b.id("bytes"), "length"),
				),
			),
		),
		b.panic("unsafe codec access is out of range"),
		nil,
	)
}

func (b builder) method(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.PublicKeyword()},
		nil,
		b.id(name),
		nil,
		nil,
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}

func (b builder) typeParameter(name string) tsgo.TypeParameterDeclaration {
	return b.factory.TypeParameterDeclaration(nil, b.id(name), nil, nil, nil)
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

func (b builder) variable(
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
			tsgo.NodeFlagsConst,
		),
	)
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

func (b builder) panic(message string) tsgo.Statement {
	return b.factory.ExpressionStatement(panicruntime.Call(
		b.factory,
		b.panicName,
		b.factory.StringLiteral(message, tsgo.TokenFlagsNone),
	))
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

func (b builder) typeReference(name string) tsgo.TypeReferenceNode {
	return b.factory.TypeReferenceNode(b.id(name), nil)
}

func (b builder) byteArrayType() tsgo.TypeNode {
	return b.typeReference("Uint8Array")
}

func (b builder) numberType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword)
}

func (b builder) voidType() tsgo.TypeNode {
	return b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword)
}

func (b builder) id(name string) tsgo.Identifier {
	return b.factory.Identifier(name)
}

func (b builder) number(value string) tsgo.NumericLiteral {
	return b.factory.NumericLiteral(value, tsgo.TokenFlagsNone)
}
