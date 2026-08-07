package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) storageMethods() []tsgo.ClassElement {
	return []tsgo.ClassElement{
		b.allocateMethod(),
		b.grownCapacityMethod(),
		b.initializeMethod(),
		b.withLengthMethod(),
	}
}

func (b builder) allocateMethod() tsgo.MethodDeclaration {
	numericLength := b.id("numericLength")
	resolvedCapacity := b.id("resolvedCapacity")
	capacity := b.id("capacity")
	invalid := b.binary(
		b.binary(
			numericLength,
			tsgo.BinaryOperatorLessThanToken,
			b.number("0"),
		),
		tsgo.BinaryOperatorBarBarToken,
		b.binary(
			resolvedCapacity,
			tsgo.BinaryOperatorLessThanToken,
			numericLength,
		),
	)
	array := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{resolvedCapacity},
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		StorageAllocateMember,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter("capacity", b.optionalIntegerInputType()),
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
			b.resolvedCapacity(capacity, numericLength),
		),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.returnStatement(b.newSlice(
			array,
			b.number("0"),
			numericLength,
			resolvedCapacity,
		)),
	)
}

func (b builder) grownCapacityMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		StorageGrownCapacityMember,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("capacity", b.numberType()),
			b.parameter("length", b.numberType()),
		},
		b.numberType(),
		b.variable(
			tsgo.NodeFlagsLet,
			"nextCapacity",
			b.initialGrowthCapacity(b.id("capacity")),
		),
		b.growCapacityLoop(b.id("length")),
		b.returnStatement(b.id("nextCapacity")),
	)
}

func (b builder) initializeMethod() tsgo.MethodDeclaration {
	invalid := b.binary(
		b.binary(
			b.id("backing"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.factory.NullLiteral(),
		),
		tsgo.BinaryOperatorBarBarToken,
		b.binary(
			b.binary(
				b.id("index"),
				tsgo.BinaryOperatorLessThanToken,
				b.number("0"),
			),
			tsgo.BinaryOperatorBarBarToken,
			b.binary(
				b.id("index"),
				tsgo.BinaryOperatorGreaterThanEqualsToken,
				b.thisProperty(MemberName(MemberCapacity)),
			),
		),
	)
	return b.method(
		nil,
		StorageInitializeMember,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("index", b.numberType()),
			b.parameter("value", b.typeT()),
		},
		b.factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.thisProperty("backing"),
		),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.factory.ExpressionStatement(
			b.assign(
				b.backingElement(b.id("backing"), b.id("index")),
				b.id("value"),
			),
		),
	)
}

func (b builder) withLengthMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		StorageWithLengthMember,
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("length", b.numberType())},
		b.sliceType(),
		b.returnStatement(b.call(
			b.factory.ThisExpression(),
			MemberName(MemberSlice),
			b.number("0"),
			b.id("length"),
			b.factory.NullLiteral(),
		)),
	)
}
