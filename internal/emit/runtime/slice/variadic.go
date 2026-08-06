package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) appendSliceMethod() tsgo.MethodDeclaration {
	source := b.id("source")
	values := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.property(source, MemberName(MemberLength))},
	)
	capture := b.loop(
		b.property(source, MemberName(MemberLength)),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(b.id("values"), b.id("index")),
				b.call(source, MemberName(MemberGet), b.id("index")),
			),
		),
	)
	newLength := b.add(
		b.thisProperty(MemberName(MemberLength)),
		b.property(b.id("values"), "length"),
	)
	reuse := b.factory.Block([]tsgo.Statement{
		b.factory.IfStatement(
			b.binary(
				b.id("existingBacking"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.factory.NullLiteral(),
			),
			b.throwBounds(),
			nil,
		),
		b.loop(
			b.property(b.id("values"), "length"),
			b.factory.ExpressionStatement(
				b.assign(
					b.index(
						b.id("existingBacking"),
						b.add(
							b.add(
								b.thisProperty("offset"),
								b.thisProperty(MemberName(MemberLength)),
							),
							b.id("index"),
						),
					),
					b.indexedValue(b.id("values"), b.id("index")),
				),
			),
		),
		b.returnStatement(b.newSlice(
			b.id("existingBacking"),
			b.thisProperty("offset"),
			b.id("newLength"),
			b.thisProperty(MemberName(MemberCapacity)),
		)),
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
			b.factory.ExpressionStatement(b.assign(
				b.id("nextCapacity"),
				b.binary(
					b.id("nextCapacity"),
					tsgo.BinaryOperatorAsteriskToken,
					b.number("2"),
				),
			)),
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
				b.indexedValue(
					b.id("existingBacking"),
					b.add(b.thisProperty("offset"), b.id("index")),
				),
			),
		),
	)
	copyAppended := b.loop(
		b.property(b.id("values"), "length"),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(
					b.id("backing"),
					b.add(
						b.thisProperty(MemberName(MemberLength)),
						b.id("index"),
					),
				),
				b.indexedValue(b.id("values"), b.id("index")),
			),
		),
	)
	return b.method(
		nil,
		MemberName(MemberAppendSlice),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.typeT()),
			b.parameter("source", b.sliceType()),
		},
		b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "values", values),
		capture,
		b.variable(tsgo.NodeFlagsConst, "newLength", newLength),
		b.variable(
			tsgo.NodeFlagsConst,
			"existingBacking",
			b.thisProperty("backing"),
		),
		b.factory.IfStatement(
			b.binary(
				b.property(b.id("values"), "length"),
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				b.number("0"),
			),
			b.returnStatement(b.factory.ThisExpression()),
			nil,
		),
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
			b.call(backing, "fill", b.id("zero")),
		),
		b.factory.IfStatement(
			b.binary(
				b.id("existingBacking"),
				tsgo.BinaryOperatorExclamationEqualsEqualsToken,
				b.factory.NullLiteral(),
			),
			b.factory.Block([]tsgo.Statement{copyExisting}, true),
			nil,
		),
		copyAppended,
		b.returnStatement(b.newSlice(
			b.id("backing"),
			b.number("0"),
			b.id("newLength"),
			b.id("nextCapacity"),
		)),
	)
}
