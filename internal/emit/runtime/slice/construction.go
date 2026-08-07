package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) constructor() tsgo.ConstructorDeclaration {
	privateReadonly := []tsgo.ModifierLike{
		b.factory.PrivateKeyword(),
		b.factory.ReadonlyKeyword(),
	}
	publicReadonly := []tsgo.ModifierLike{b.factory.ReadonlyKeyword()}
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.ProtectedKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				privateReadonly,
				nil,
				b.id("backing"),
				nil,
				b.backingType(),
				nil,
			),
			b.factory.ParameterDeclaration(
				privateReadonly,
				nil,
				b.id("offset"),
				nil,
				b.numberType(),
				nil,
			),
			b.factory.ParameterDeclaration(
				publicReadonly,
				nil,
				b.id(MemberName(MemberLength)),
				nil,
				b.numberType(),
				nil,
			),
			b.factory.ParameterDeclaration(
				publicReadonly,
				nil,
				b.id(MemberName(MemberCapacity)),
				nil,
				b.numberType(),
				nil,
			),
		},
		nil,
		b.factory.Block(nil, true),
	)
}

func (b builder) nilMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		MemberName(MemberNil),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		nil,
		b.sliceType(),
		b.returnStatement(
			b.newSlice(
				b.factory.NullLiteral(),
				b.number("0"),
				b.number("0"),
				b.number("0"),
			),
		),
	)
}

func (b builder) makeMethod() tsgo.MethodDeclaration {
	invalidLength := b.binary(
		b.id("numericLength"),
		tsgo.BinaryOperatorLessThanToken,
		b.number("0"),
	)
	capacityBelowLength := b.binary(
		b.id("resolvedCapacity"),
		tsgo.BinaryOperatorLessThanToken,
		b.id("numericLength"),
	)
	invalid := b.binary(
		invalidLength,
		tsgo.BinaryOperatorBarBarToken,
		capacityBelowLength,
	)
	array := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("resolvedCapacity")},
	)
	resolvedCapacity := b.resolvedCapacity(
		b.id("capacity"),
		b.id("numericLength"),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		MemberName(MemberMake),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter(
				"capacity",
				b.factory.UnionTypeNode([]tsgo.TypeNode{
					b.integerInputType(),
					b.factory.LiteralTypeNode(b.factory.NullLiteral()),
				}),
			),
			b.parameter("zero", b.typeT()),
		},
		b.sliceType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"numericLength",
			b.toNumber(b.id("length")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"resolvedCapacity",
			resolvedCapacity,
		),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.call(array, "fill", b.id("zero")),
		),
		b.returnStatement(
			b.newSlice(
				b.id("backing"),
				b.number("0"),
				b.id("numericLength"),
				b.id("resolvedCapacity"),
			),
		),
	)
}

func (b builder) literalMethod() tsgo.MethodDeclaration {
	length := b.property(b.id("values"), "length")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		MemberName(MemberLiteral),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("values", b.factory.ArrayTypeNode(b.typeT())),
		},
		b.sliceType(),
		b.returnStatement(
			b.newSlice(
				b.id("values"),
				b.number("0"),
				length,
				length,
			),
		),
	)
}

func (b builder) isNilMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		MemberName(MemberIsNil),
		nil,
		nil,
		b.booleanType(),
		b.returnStatement(
			b.binary(
				b.thisProperty("backing"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.factory.NullLiteral(),
			),
		),
	)
}
