package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) appendMethod() tsgo.MethodDeclaration {
	values := b.factory.ParameterDeclaration(
		nil,
		b.factory.DotDotDotToken(),
		b.id("values"),
		nil,
		b.factory.ArrayTypeNode(b.typeT()),
		nil,
	)
	newLength := b.add(
		b.thisProperty(MemberName(MemberLength)),
		b.property(b.id("values"), "length"),
	)
	reuseLoop := b.loop(
		b.property(b.id("values"), "length"),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(
					b.factory.NonNullExpression(
						b.thisProperty("backing"),
						tsgo.NodeFlagsNone,
					),
					b.add(
						b.add(
							b.thisProperty("offset"),
							b.thisProperty(MemberName(MemberLength)),
						),
						b.id("index"),
					),
				),
				b.index(b.id("values"), b.id("index")),
			),
		),
	)
	reuse := b.factory.Block([]tsgo.Statement{
		reuseLoop,
		b.returnStatement(
			b.newSlice(
				b.thisProperty("backing"),
				b.thisProperty("offset"),
				b.id("newLength"),
				b.thisProperty(MemberName(MemberCapacity)),
				b.thisProperty("zero"),
			),
		),
	}, true)
	initialCapacity := b.factory.ConditionalExpression(
		b.binary(
			b.thisProperty(MemberName(MemberCapacity)),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.number("0"),
		),
		b.factory.QuestionToken(),
		b.number("1"),
		b.factory.ColonToken(),
		b.binary(
			b.thisProperty(MemberName(MemberCapacity)),
			tsgo.BinaryOperatorAsteriskToken,
			b.number("2"),
		),
	)
	growCapacity := b.factory.WhileStatement(
		b.binary(
			b.id("nextCapacity"),
			tsgo.BinaryOperatorLessThanToken,
			b.id("newLength"),
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
	backing := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("nextCapacity")},
	)
	copyExisting := b.loop(
		b.thisProperty(MemberName(MemberLength)),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(b.id("backing"), b.id("index")),
				b.backingElement(b.id("index")),
			),
		),
	)
	copyAppended := b.factory.ForStatement(
		b.factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				b.factory.VariableDeclaration(
					b.id("index"),
					nil,
					nil,
					b.number("0"),
				),
			},
			tsgo.NodeFlagsLet,
		),
		b.binary(
			b.id("index"),
			tsgo.BinaryOperatorLessThanToken,
			b.property(b.id("values"), "length"),
		),
		b.assign(
			b.id("index"),
			b.add(b.id("index"), b.number("1")),
		),
		b.factory.Block([]tsgo.Statement{
			b.factory.ExpressionStatement(
				b.assign(
					b.index(
						b.id("backing"),
						b.add(
							b.thisProperty(MemberName(MemberLength)),
							b.id("index"),
						),
					),
					b.index(b.id("values"), b.id("index")),
				),
			),
		}, true),
	)
	return b.method(
		nil,
		MemberName(MemberAppend),
		nil,
		[]tsgo.ParameterDeclaration{values},
		b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "newLength", newLength),
		b.factory.IfStatement(
			b.binary(
				b.id("newLength"),
				tsgo.BinaryOperatorLessThanEqualsToken,
				b.thisProperty(MemberName(MemberCapacity)),
			),
			reuse,
			nil,
		),
		b.variable(tsgo.NodeFlagsLet, "nextCapacity", initialCapacity),
		growCapacity,
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.call(backing, "fill", b.thisProperty("zero")),
		),
		copyExisting,
		copyAppended,
		b.returnStatement(
			b.newSlice(
				b.id("backing"),
				b.number("0"),
				b.id("newLength"),
				b.id("nextCapacity"),
				b.thisProperty("zero"),
			),
		),
	)
}

func (b builder) copyMethod() tsgo.MethodDeclaration {
	count := b.factory.CallExpression(
		b.property(b.id("Math"), "min"),
		nil,
		nil,
		[]tsgo.Expression{
			b.property(b.id("target"), MemberName(MemberLength)),
			b.property(b.id("source"), MemberName(MemberLength)),
		},
		tsgo.NodeFlagsNone,
	)
	sameBacking := b.binary(
		b.property(b.id("target"), "backing"),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.property(b.id("source"), "backing"),
	)
	copyWithin := b.call(
		b.factory.NonNullExpression(
			b.property(b.id("target"), "backing"),
			tsgo.NodeFlagsNone,
		),
		"copyWithin",
		b.property(b.id("target"), "offset"),
		b.property(b.id("source"), "offset"),
		b.add(b.property(b.id("source"), "offset"), b.id("count")),
	)
	distinctCopy := b.loop(
		b.id("count"),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(
					b.factory.NonNullExpression(
						b.property(b.id("target"), "backing"),
						tsgo.NodeFlagsNone,
					),
					b.add(
						b.property(b.id("target"), "offset"),
						b.id("index"),
					),
				),
				b.index(
					b.factory.NonNullExpression(
						b.property(b.id("source"), "backing"),
						tsgo.NodeFlagsNone,
					),
					b.add(
						b.property(b.id("source"), "offset"),
						b.id("index"),
					),
				),
			),
		),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		MemberName(MemberCopy),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("target", b.sliceType()),
			b.parameter("source", b.sliceType()),
		},
		b.numberType(),
		b.variable(tsgo.NodeFlagsConst, "count", count),
		b.factory.IfStatement(
			b.binary(
				b.id("count"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.number("0"),
			),
			b.returnStatement(b.number("0")),
			nil,
		),
		b.factory.IfStatement(
			sameBacking,
			b.factory.ExpressionStatement(copyWithin),
			distinctCopy,
		),
		b.returnStatement(b.id("count")),
	)
}
