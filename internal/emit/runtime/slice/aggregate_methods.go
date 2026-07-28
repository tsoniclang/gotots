package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const (
	aggregateShapeMember   = "$shape"
	aggregateGrowthMember  = "$grownCapacity"
	aggregateNilMember     = "nilWith"
	aggregateMakeMember    = "makeWith"
	aggregateLiteralMember = "literalWith"
	aggregateAppendMember  = "appendWith"
	aggregateCopyMember    = "copyWith"
)

func (b builder) shapeMethod() tsgo.MethodDeclaration {
	numericLength := b.toNumber(b.id("length"))
	resolvedCapacity := b.factory.ConditionalExpression(
		b.binary(
			b.id("capacity"),
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			b.factory.NullLiteral(),
		),
		b.factory.QuestionToken(),
		b.id("numericLength"),
		b.factory.ColonToken(),
		b.toNumber(b.id("capacity")),
	)
	invalid := b.binary(
		b.binary(
			b.id("numericLength"),
			tsgo.BinaryOperatorLessThanToken,
			b.number("0"),
		),
		tsgo.BinaryOperatorBarBarToken,
		b.binary(
			b.id("resolvedCapacity"),
			tsgo.BinaryOperatorLessThanToken,
			b.id("numericLength"),
		),
	)
	return b.method(
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		aggregateShapeMember,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter("capacity", b.optionalIntegerInputType()),
		},
		b.factory.TupleTypeNode([]tsgo.TypeNode{
			b.numberType(),
			b.numberType(),
		}),
		b.variable(tsgo.NodeFlagsConst, "numericLength", numericLength),
		b.variable(
			tsgo.NodeFlagsConst,
			"resolvedCapacity",
			resolvedCapacity,
		),
		b.factory.IfStatement(invalid, b.throwBounds(), nil),
		b.returnStatement(b.factory.ArrayLiteralExpression(
			[]tsgo.Expression{
				b.id("numericLength"),
				b.id("resolvedCapacity"),
			},
			false,
		)),
	)
}

func (b builder) makeMethodWithShape(lazyZero bool) tsgo.MethodDeclaration {
	shape := b.call(
		b.id(b.className),
		aggregateShapeMember,
		b.id("length"),
		b.id("capacity"),
	)
	backing := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("resolvedCapacity")},
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		MemberName(MemberMake),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter("capacity", b.optionalIntegerInputType()),
			b.parameter("zero", b.typeT()),
		},
		b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "shape", shape),
		b.variable(
			tsgo.NodeFlagsConst,
			"numericLength",
			b.index(b.id("shape"), b.number("0")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"resolvedCapacity",
			b.index(b.id("shape"), b.number("1")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"backing",
			b.call(backing, "fill", b.id("zero")),
		),
		b.returnStatement(
			b.newSlice(
				b.id("backing"),
				b.number("0"),
				b.id("numericLength"),
				b.id("resolvedCapacity"),
				b.zeroStorage(lazyZero, b.id("zero")),
			),
		),
	)
}

func (b builder) aggregateNilMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		aggregateNilMember,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.valueFactoryType()),
		},
		b.sliceType(),
		b.returnStatement(
			b.newSlice(
				b.factory.NullLiteral(),
				b.number("0"),
				b.number("0"),
				b.number("0"),
				b.id("zero"),
			),
		),
	)
}

func (b builder) aggregateMakeMethod() tsgo.MethodDeclaration {
	shape := b.call(
		b.id(b.className),
		aggregateShapeMember,
		b.id("length"),
		b.id("capacity"),
	)
	backing := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("resolvedCapacity")},
	)
	fill := b.loop(
		b.id("resolvedCapacity"),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(b.id("backing"), b.id("index")),
				b.invoke(b.id("zero")),
			),
		),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		aggregateMakeMember,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("length", b.integerInputType()),
			b.parameter("capacity", b.optionalIntegerInputType()),
			b.parameter("zero", b.valueFactoryType()),
		},
		b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "shape", shape),
		b.variable(
			tsgo.NodeFlagsConst,
			"numericLength",
			b.index(b.id("shape"), b.number("0")),
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"resolvedCapacity",
			b.index(b.id("shape"), b.number("1")),
		),
		b.variable(tsgo.NodeFlagsConst, "backing", backing),
		fill,
		b.returnStatement(
			b.newSlice(
				b.id("backing"),
				b.number("0"),
				b.id("numericLength"),
				b.id("resolvedCapacity"),
				b.id("zero"),
			),
		),
	)
}

func (b builder) aggregateLiteralMethod() tsgo.MethodDeclaration {
	values := b.factory.ParameterDeclaration(
		nil,
		b.factory.DotDotDotToken(),
		b.id("values"),
		nil,
		b.factory.ArrayTypeNode(b.typeT()),
		nil,
	)
	length := b.property(b.id("values"), "length")
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		aggregateLiteralMember,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.valueFactoryType()),
			values,
		},
		b.sliceType(),
		b.returnStatement(
			b.newSlice(
				b.id("values"),
				b.number("0"),
				length,
				length,
				b.id("zero"),
			),
		),
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
		[]tsgo.ModifierLike{
			b.factory.PrivateKeyword(),
			b.factory.StaticKeyword(),
		},
		aggregateGrowthMember,
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

func (b builder) aggregateAppendMethod() tsgo.MethodDeclaration {
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
	reuse := b.factory.ReturnStatement(
		b.factory.CallExpression(
			b.property(b.factory.ThisExpression(), MemberName(MemberAppend)),
			nil,
			nil,
			[]tsgo.Expression{b.factory.SpreadElement(b.id("values"))},
			tsgo.NodeFlagsNone,
		),
	)
	nextCapacity := b.call(
		b.id(b.className),
		aggregateGrowthMember,
		b.thisProperty(MemberName(MemberCapacity)),
		b.id("newLength"),
	)
	backing := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("nextCapacity")},
	)
	fill := b.loop(
		b.id("nextCapacity"),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(b.id("backing"), b.id("index")),
				b.invoke(b.id("zero")),
			),
		),
	)
	copyExisting := b.loop(
		b.thisProperty(MemberName(MemberLength)),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(b.id("backing"), b.id("index")),
				b.invoke(
					b.id("copyValue"),
					b.index(
						b.id("existingBacking"),
						b.add(b.thisProperty("offset"), b.id("index")),
					),
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
				b.index(b.id("values"), b.id("index")),
			),
		),
	)
	return b.method(
		nil,
		aggregateAppendMember,
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.valueFactoryType()),
			b.parameter("copyValue", b.valueCopyType()),
			values,
		},
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
		b.variable(
			tsgo.NodeFlagsConst,
			"existingBacking",
			b.thisProperty("backing"),
		),
		b.variable(tsgo.NodeFlagsConst, "nextCapacity", nextCapacity),
		b.variable(tsgo.NodeFlagsConst, "backing", backing),
		fill,
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
				b.id("zero"),
			),
		),
	)
}

func (b builder) aggregateCopyMethod() tsgo.MethodDeclaration {
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
	snapshot := b.factory.NewExpression(
		b.id("Array"),
		[]tsgo.TypeNode{b.typeT()},
		[]tsgo.Expression{b.id("count")},
	)
	capture := b.loop(
		b.id("count"),
		b.factory.ExpressionStatement(
			b.assign(
				b.index(b.id("snapshot"), b.id("index")),
				b.invoke(
					b.id("copyValue"),
					b.call(
						b.id("source"),
						MemberName(MemberGet),
						b.id("index"),
					),
				),
			),
		),
	)
	store := b.loop(
		b.id("count"),
		b.factory.ExpressionStatement(
			b.call(
				b.id("target"),
				MemberName(MemberSet),
				b.id("index"),
				b.index(b.id("snapshot"), b.id("index")),
			),
		),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		aggregateCopyMember,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("target", b.sliceType()),
			b.parameter("source", b.sliceType()),
			b.parameter("copyValue", b.valueCopyType()),
		},
		b.numberType(),
		b.variable(tsgo.NodeFlagsConst, "count", count),
		b.variable(tsgo.NodeFlagsConst, "snapshot", snapshot),
		capture,
		store,
		b.returnStatement(b.id("count")),
	)
}

func (b builder) valueFactoryType() tsgo.FunctionTypeNode {
	return b.factory.FunctionTypeNode(nil, nil, b.typeT())
}

func (b builder) valueCopyType() tsgo.FunctionTypeNode {
	return b.factory.FunctionTypeNode(
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
		},
		b.typeT(),
	)
}

func (b builder) invoke(
	callee tsgo.Expression,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return b.factory.CallExpression(
		callee,
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}
