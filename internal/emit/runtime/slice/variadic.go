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
	return b.method(
		nil,
		MemberName(MemberAppendSlice),
		nil,
		[]tsgo.ParameterDeclaration{b.parameter("source", b.sliceType())},
		b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "values", values),
		capture,
		b.returnStatement(
			b.call(
				b.factory.ThisExpression(),
				MemberName(MemberAppend),
				b.factory.SpreadElement(b.id("values")),
			),
		),
	)
}

func (b builder) aggregateAppendSliceMethod() tsgo.MethodDeclaration {
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
				b.invoke(
					b.id("copyValue"),
					b.call(source, MemberName(MemberGet), b.id("index")),
				),
			),
		),
	)
	return b.method(
		nil,
		MemberName(MemberAppendSliceWith),
		nil,
		[]tsgo.ParameterDeclaration{
			b.parameter("zero", b.valueFactoryType()),
			b.parameter("copyValue", b.valueCopyType()),
			b.parameter("source", b.sliceType()),
		},
		b.sliceType(),
		b.variable(tsgo.NodeFlagsConst, "values", values),
		capture,
		b.returnStatement(
			b.call(
				b.factory.ThisExpression(),
				aggregateAppendMember,
				b.id("zero"),
				b.id("copyValue"),
				b.factory.SpreadElement(b.id("values")),
			),
		),
	)
}
