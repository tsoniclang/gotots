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
			b.factory.ConditionalExpression(
				b.binary(
					capacity,
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
				b.factory.QuestionToken(),
				numericLength,
				b.factory.ColonToken(),
				b.toNumber(capacity),
			),
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
	initial := b.factory.ConditionalExpression(
		b.binary(
			b.id("capacity"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.number("0"),
		),
		b.factory.QuestionToken(),
		b.number("1"),
		b.factory.ColonToken(),
		b.binary(
			b.id("capacity"),
			tsgo.BinaryOperatorAsteriskToken,
			b.number("2"),
		),
	)
	grow := b.factory.WhileStatement(
		b.binary(
			b.id("nextCapacity"),
			tsgo.BinaryOperatorLessThanToken,
			b.id("length"),
		),
		b.factory.Block([]tsgo.Statement{
			b.factory.ExpressionStatement(
				b.assign(
					b.id("nextCapacity"),
					b.binary(
						b.id("nextCapacity"),
						tsgo.BinaryOperatorAsteriskToken,
						b.number("2"),
					),
				),
			),
		}, true),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		StorageGrownCapacityMember,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("capacity", b.numberType()),
			b.parameter("length", b.numberType()),
		},
		b.numberType(),
		b.variable(tsgo.NodeFlagsLet, "nextCapacity", initial),
		grow,
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
	invalid := b.binary(
		b.binary(
			b.id("length"),
			tsgo.BinaryOperatorLessThanToken,
			b.number("0"),
		),
		tsgo.BinaryOperatorBarBarToken,
		b.binary(
			b.id("length"),
			tsgo.BinaryOperatorGreaterThanToken,
			b.thisProperty(MemberName(MemberCapacity)),
		),
	)
	return b.method(
		nil,
		StorageWithLengthMember,
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("length", b.numberType())},
		b.sliceType(),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.returnStatement(b.newSlice(
			b.thisProperty("backing"),
			b.thisProperty("offset"),
			b.id("length"),
			b.thisProperty(MemberName(MemberCapacity)),
		)),
	)
}
