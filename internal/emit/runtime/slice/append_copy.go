package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) appendMethod(sharedGrowth bool) tsgo.MethodDeclaration {
	newLength := b.add(
		b.thisProperty(MemberName(MemberLength)),
		b.property(b.id("values"), "length"),
	)
	zero := tsgo.Expression(b.id("zero"))
	reuseLoop := b.loop(
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
		reuseLoop,
		b.returnStatement(
			b.newSlice(
				b.id("existingBacking"),
				b.thisProperty("offset"),
				b.id("newLength"),
				b.thisProperty(MemberName(MemberCapacity)),
			),
		),
	}, true)
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
	statements := []tsgo.Statement{
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
	}
	if sharedGrowth {
		statements = append(
			statements,
			b.variable(
				tsgo.NodeFlagsConst,
				"nextCapacity",
				b.call(
					b.id(b.className),
					StorageGrownCapacityMember,
					b.thisProperty(MemberName(MemberCapacity)),
					b.id("newLength"),
				),
			),
		)
	} else {
		statements = append(
			statements,
			b.variable(
				tsgo.NodeFlagsLet,
				"nextCapacity",
				b.initialGrowthCapacity(
					b.thisProperty(MemberName(MemberCapacity)),
				),
			),
			b.growCapacityLoop(b.id("newLength")),
		)
	}
	statements = append(
		statements,
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.call(backing, "fill", zero),
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
		b.returnStatement(
			b.newSlice(
				b.id("backing"),
				b.number("0"),
				b.id("newLength"),
				b.id("nextCapacity"),
			),
		),
	)
	return b.method(
		nil,
		MemberName(MemberAppend),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.typeT()),
			b.parameter("values", b.factory.ArrayTypeNode(b.typeT())),
		},
		b.sliceType(),
		statements...,
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
		b.id("targetBacking"),
		tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		b.id("sourceBacking"),
	)
	copyWithin := b.call(
		b.id("targetBacking"),
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
					b.id("targetBacking"),
					b.add(
						b.property(b.id("target"), "offset"),
						b.id("index"),
					),
				),
				b.indexedValue(
					b.id("sourceBacking"),
					b.add(
						b.property(b.id("source"), "offset"),
						b.id("index"),
					),
				),
			),
		),
	)
	directCopy := b.factory.Block([]tsgo.Statement{
		b.factory.IfStatement(
			sameBacking,
			b.factory.ExpressionStatement(copyWithin),
			distinctCopy,
		),
		b.returnStatement(b.id("count")),
	}, true)
	values := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("count")},
	)
	captureProjected := b.loop(
		b.id("count"),
		b.factory.ExpressionStatement(b.assign(
			b.index(b.id("values"), b.id("index")),
			b.call(b.id("source"), MemberName(MemberGet), b.id("index")),
		)),
	)
	writeProjected := b.loop(
		b.id("count"),
		b.factory.ExpressionStatement(b.call(
			b.id("target"),
			MemberName(MemberSet),
			b.id("index"),
			b.indexedValue(b.id("values"), b.id("index")),
		)),
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
		b.variable(
			tsgo.NodeFlagsConst,
			"targetBacking",
			b.property(b.id("target"), "backing"),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"sourceBacking",
			b.property(b.id("source"), "backing"),
		),
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
			b.binary(
				b.binary(
					b.id("targetBacking"),
					tsgo.BinaryOperatorExclamationEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
				tsgo.BinaryOperatorAmpersandAmpersandToken,
				b.binary(
					b.id("sourceBacking"),
					tsgo.BinaryOperatorExclamationEqualsEqualsToken,
					b.factory.NullLiteral(),
				),
			),
			directCopy,
			nil,
		),
		b.variable(tsgo.NodeFlagsConst, "values", values),
		captureProjected,
		writeProjected,
		b.returnStatement(b.id("count")),
	)
}
