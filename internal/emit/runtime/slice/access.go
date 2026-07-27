package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) boundsCondition(index tsgo.Expression) tsgo.Expression {
	belowZero := b.binary(
		index,
		tsgo.BinaryOperatorLessThanToken,
		b.number("0"),
	)
	atOrAboveLength := b.binary(
		index,
		tsgo.BinaryOperatorGreaterThanEqualsToken,
		b.thisProperty(MemberName(MemberLength)),
	)
	return b.binary(
		belowZero,
		tsgo.BinaryOperatorBarBarToken,
		atOrAboveLength,
	)
}

func (b builder) backingElement(
	backing tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return b.index(
		backing,
		b.add(b.thisProperty("offset"), index),
	)
}

func (b builder) getMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		MemberName(MemberGet),
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("index", b.integerInputType())},
		b.typeT(),
		b.variable(
			tsgo.NodeFlagsConst,
			"numericIndex",
			b.toNumber(b.id("index")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.thisProperty("backing"),
		),
		b.factory.IfStatement(
			b.binary(
				b.binary(
					b.id("backing"),
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
				tsgo.BinaryOperatorBarBarToken,
				b.boundsCondition(b.id("numericIndex")),
			),
			b.throwBounds(),
			nil,
		),
		b.returnStatement(
			b.backingElement(b.id("backing"), b.id("numericIndex")),
		),
	)
}

func (b builder) setMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		MemberName(MemberSet),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("index", b.integerInputType()),
			b.parameter("value", b.typeT()),
		},
		b.typeT(),
		b.variable(
			tsgo.NodeFlagsConst,
			"numericIndex",
			b.toNumber(b.id("index")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.thisProperty("backing"),
		),
		b.factory.IfStatement(
			b.binary(
				b.binary(
					b.id("backing"),
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
				tsgo.BinaryOperatorBarBarToken,
				b.boundsCondition(b.id("numericIndex")),
			),
			b.throwBounds(),
			nil,
		),
		b.factory.ExpressionStatement(
			b.assign(
				b.backingElement(b.id("backing"), b.id("numericIndex")),
				b.id("value"),
			),
		),
		b.returnStatement(b.id("value")),
	)
}

func (b builder) sliceMethod() tsgo.MethodDeclaration {
	resolvedHigh := b.factory.ConditionalExpression(
		b.binary(
			b.id("high"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.factory.NullLiteral(),
		),
		b.factory.QuestionToken(),
		b.thisProperty(MemberName(MemberLength)),
		b.factory.ColonToken(),
		b.toNumber(b.id("high")),
	)
	resolvedMax := b.factory.ConditionalExpression(
		b.binary(
			b.id("max"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.factory.NullLiteral(),
		),
		b.factory.QuestionToken(),
		b.thisProperty(MemberName(MemberCapacity)),
		b.factory.ColonToken(),
		b.toNumber(b.id("max")),
	)
	invalid := b.binary(
		b.binary(
			b.binary(
				b.id("numericLow"),
				tsgo.BinaryOperatorLessThanToken,
				b.number("0"),
			),
			tsgo.BinaryOperatorBarBarToken,
			b.binary(
				b.id("resolvedHigh"),
				tsgo.BinaryOperatorLessThanToken,
				b.id("numericLow"),
			),
		),
		tsgo.BinaryOperatorBarBarToken,
		b.binary(
			b.binary(
				b.id("resolvedMax"),
				tsgo.BinaryOperatorLessThanToken,
				b.id("resolvedHigh"),
			),
			tsgo.BinaryOperatorBarBarToken,
			b.binary(
				b.id("resolvedMax"),
				tsgo.BinaryOperatorGreaterThanToken,
				b.thisProperty(MemberName(MemberCapacity)),
			),
		),
	)
	return b.method(
		nil,
		MemberName(MemberSlice),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("low", b.integerInputType()),
			b.parameter("high", b.optionalIntegerInputType()),
			b.parameter("max", b.optionalIntegerInputType()),
		},
		b.sliceType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"numericLow",
			b.toNumber(b.id("low")),
		),
		b.variable(tsgo.NodeFlagsConst, "resolvedHigh", resolvedHigh),
		b.variable(tsgo.NodeFlagsConst, "resolvedMax", resolvedMax),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.returnStatement(
			b.newSlice(
				b.thisProperty("backing"),
				b.add(b.thisProperty("offset"), b.id("numericLow")),
				b.subtract(b.id("resolvedHigh"), b.id("numericLow")),
				b.subtract(b.id("resolvedMax"), b.id("numericLow")),
				b.thisProperty("zero"),
			),
		),
	)
}
