package unsafepointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) integerType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.numberType(),
		b.bigintType(),
	})
}

func (b builder) fromInteger() tsgo.MethodDeclaration {
	value := b.id("value")
	zero := b.id("zero")
	numeric := b.id("numeric")
	allocation := b.id("allocation")
	offset := b.id("offset")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PublicKeyword(), b.factory.StaticKeyword()},
		FromIntegerName,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.integerType(), nil),
			b.parameter("zero", b.integerType(), nil),
		},
		b.optional(b.unsafeType()),
		b.factory.IfStatement(
			b.binary(value, tsgo.BinaryOperatorEqualsEqualsEqualsToken, zero),
			b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(b.undefined())}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"numeric",
			b.numberType(),
			b.factory.CallExpression(
				b.property(b.id("globalThis"), "Number"),
				nil,
				nil,
				[]tsgo.Expression{value},
				tsgo.NodeFlagsNone,
			),
		),
		b.factory.IfStatement(
			b.notSafeInteger(numeric),
			b.panic("unsafe integer address is not representable"),
			nil,
		),
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{b.factory.VariableDeclaration(
					allocation,
					nil,
					nil,
					nil,
				)},
				tsgo.NodeFlagsConst,
			),
			b.property(b.id(b.className), "allocations"),
			b.factory.Block([]tsgo.Statement{
				b.variable(
					tsgo.NodeFlagsConst,
					"offset",
					b.numberType(),
					b.binary(
						numeric,
						tsgo.BinaryOperatorMinusToken,
						b.property(allocation, "base"),
					),
				),
				b.factory.IfStatement(
					b.binary(
						b.binary(
							offset,
							tsgo.BinaryOperatorGreaterThanEqualsToken,
							b.number("0"),
						),
						tsgo.BinaryOperatorAmpersandAmpersandToken,
						b.binary(
							offset,
							tsgo.BinaryOperatorLessThanEqualsToken,
							b.property(allocation, "length"),
						),
					),
					b.factory.Block([]tsgo.Statement{
						b.factory.ReturnStatement(b.call(allocation, atName, offset)),
					}, true),
					nil,
				),
			}, true),
		),
		b.panic("unsafe integer address does not identify live generated memory"),
	)
}

func (b builder) toIntegerNumberOverload() tsgo.MethodDeclaration {
	return b.toIntegerOverload(b.numberType())
}

func (b builder) toIntegerBigIntOverload() tsgo.MethodDeclaration {
	return b.toIntegerOverload(b.bigintType())
}

func (b builder) toIntegerOverload(target tsgo.TypeNode) tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.PublicKeyword(), b.factory.StaticKeyword()},
		ToIntegerName,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.optional(b.unsafeType()), nil),
			b.parameter("zero", target, nil),
		},
		target,
	)
}

func (b builder) toInteger() tsgo.MethodDeclaration {
	value := b.id("value")
	zero := b.id("zero")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PublicKeyword(), b.factory.StaticKeyword()},
		ToIntegerName,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.optional(b.unsafeType()), nil),
			b.parameter("zero", b.integerType(), nil),
		},
		b.integerType(),
		b.factory.IfStatement(
			b.binary(value, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{b.factory.ReturnStatement(zero)}, true),
			nil,
		),
		b.factory.ReturnStatement(b.factory.ConditionalExpression(
			b.binary(
				b.factory.TypeOfExpression(zero),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.string("bigint"),
			),
			b.factory.QuestionToken(),
			b.factory.CallExpression(
				b.property(b.id("globalThis"), "BigInt"),
				nil,
				nil,
				[]tsgo.Expression{b.property(value, addressName)},
				tsgo.NodeFlagsNone,
			),
			b.factory.ColonToken(),
			b.property(value, addressName),
		)),
	)
}
