package unsafepointer

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) integerType() tsgo.TypeNode {
	return b.factory.UnionTypeNode([]tsgo.TypeNode{
		b.numberType(),
		b.bigintType(),
	})
}

func (b builder) allocationAtAddress() tsgo.MethodDeclaration {
	value := b.id("value")
	low := b.id("low")
	high := b.id("high")
	middle := b.id("middle")
	candidate := b.id("candidate")
	allocations := b.property(b.id(b.className), "allocations")
	denseAllocation := func(index tsgo.Expression) tsgo.Expression {
		return b.call(b.id(b.denseIndexName), "get", allocations, index)
	}
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		allocationAtAddressName,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.numberType(), nil),
		},
		b.optional(b.unsafeType()),
		b.variable(tsgo.NodeFlagsLet, "low", b.numberType(), b.number("0")),
		b.variable(
			tsgo.NodeFlagsLet,
			"high",
			b.numberType(),
			b.binary(
				b.property(allocations, "length"),
				tsgo.BinaryOperatorMinusToken,
				b.number("1"),
			),
		),
		b.factory.WhileStatement(
			b.binary(low, tsgo.BinaryOperatorLessThanEqualsToken, high),
			b.factory.Block([]tsgo.Statement{
				b.variable(
					tsgo.NodeFlagsConst,
					"middle",
					b.numberType(),
					b.call(
						b.id("Math"),
						"floor",
						b.binary(
							b.binary(
								low,
								tsgo.BinaryOperatorPlusToken,
								high,
							),
							tsgo.BinaryOperatorSlashToken,
							b.number("2"),
						),
					),
				),
				b.variable(
					tsgo.NodeFlagsConst,
					"candidate",
					b.unsafeType(),
					denseAllocation(middle),
				),
				b.factory.IfStatement(
					b.binary(
						b.property(candidate, "base"),
						tsgo.BinaryOperatorLessThanEqualsToken,
						value,
					),
					b.factory.Block([]tsgo.Statement{
						b.factory.ExpressionStatement(b.assign(
							low,
							b.binary(
								middle,
								tsgo.BinaryOperatorPlusToken,
								b.number("1"),
							),
						)),
					}, true),
					b.factory.Block([]tsgo.Statement{
						b.factory.ExpressionStatement(b.assign(
							high,
							b.binary(
								middle,
								tsgo.BinaryOperatorMinusToken,
								b.number("1"),
							),
						)),
					}, true),
				),
			}, true),
		),
		b.factory.IfStatement(
			b.binary(high, tsgo.BinaryOperatorLessThanToken, b.number("0")),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(b.undefined()),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"candidate",
			b.unsafeType(),
			denseAllocation(high),
		),
		b.factory.IfStatement(
			b.binary(
				value,
				tsgo.BinaryOperatorLessThanEqualsToken,
				b.binary(
					b.property(candidate, "base"),
					tsgo.BinaryOperatorPlusToken,
					b.property(candidate, "length"),
				),
			),
			b.factory.Block([]tsgo.Statement{
				b.factory.ReturnStatement(candidate),
			}, true),
			nil,
		),
		b.factory.ReturnStatement(b.undefined()),
	)
}

func (b builder) fromRelative() tsgo.MethodDeclaration {
	value := b.id("value")
	address := b.id("address")
	zero := b.id("zero")
	numeric := b.id("numeric")
	offset := b.id("offset")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PublicKeyword(), b.factory.StaticKeyword()},
		FromRelativeName,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.optional(b.unsafeType()), nil),
			b.parameter("address", b.integerType(), nil),
			b.parameter("zero", b.integerType(), nil),
		},
		b.optional(b.unsafeType()),
		b.factory.IfStatement(
			b.binary(value, tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.undefined()),
			b.factory.Block([]tsgo.Statement{
				b.factory.IfStatement(
					b.binary(address, tsgo.BinaryOperatorEqualsEqualsEqualsToken, zero),
					b.factory.Block([]tsgo.Statement{
						b.factory.ReturnStatement(b.undefined()),
					}, true),
					b.panic("unsafe integer address does not identify live generated memory"),
				),
			}, true),
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
				[]tsgo.Expression{address},
				tsgo.NodeFlagsNone,
			),
		),
		b.factory.IfStatement(
			b.notSafeInteger(numeric),
			b.panic("unsafe integer address is not representable"),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"offset",
			b.numberType(),
			b.binary(
				numeric,
				tsgo.BinaryOperatorMinusToken,
				b.property(value, "base"),
			),
		),
		b.factory.ReturnStatement(b.call(value, atName, offset)),
	)
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
		b.variable(
			tsgo.NodeFlagsConst,
			"allocation",
			b.optional(b.unsafeType()),
			b.call(
				b.id(b.className),
				allocationAtAddressName,
				numeric,
			),
		),
		b.factory.IfStatement(
			b.binary(
				allocation,
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.undefined(),
			),
			b.panic("unsafe integer address does not identify live generated memory"),
			nil,
		),
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
		b.factory.ReturnStatement(b.call(allocation, atName, offset)),
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
