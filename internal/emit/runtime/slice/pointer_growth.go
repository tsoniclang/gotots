package slice

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b pointerSliceBuilder) pointerAppend() tsgo.MethodDeclaration {
	count := b.globalCall("BigInt", b.thisProperty("count"))
	next := b.id("nextLength")
	values := b.id("values")
	write := b.loop(b.property(values, "length"), b.factory.ExpressionStatement(b.regionCall("goRegionWrite",
		b.thisProperty("region"), b.add(count, b.globalCall("BigInt", b.id("index"))), b.indexedValue(values, b.id("index")))))
	return b.method(nil, MemberName(MemberAppend), nil, []tsgo.ParameterDeclaration{
		b.parameter("zero", b.typeT()), b.parameter("values", b.factory.ArrayTypeNode(b.typeT())),
	}, b.sliceType(),
		b.factory.IfStatement(b.binary(b.property(values, "length"), tsgo.BinaryOperatorEqualsEqualsEqualsToken, b.number("0")),
			b.returnStatement(b.factory.ThisExpression()), nil),
		b.variable(tsgo.NodeFlagsConst, "nextLength", b.add(count, b.globalCall("BigInt", b.property(values, "length")))),
		b.factory.IfStatement(b.binary(next, tsgo.BinaryOperatorLessThanEqualsToken, b.thisProperty("extent")),
			b.factory.Block([]tsgo.Statement{write, b.returnStatement(b.newPointerSlice(b.thisProperty("region"), next, b.thisProperty("extent")))}, true), nil),
		b.variable(tsgo.NodeFlagsConst, "result", b.factory.CallExpression(b.property(b.id(b.className), MemberName(MemberMake)), nil,
			[]tsgo.TypeNode{b.typeT()}, []tsgo.Expression{b.thisProperty("count"), next, b.id("zero")}, tsgo.NodeFlagsNone)),
		b.exactLoop(b.thisProperty("count"), b.factory.ExpressionStatement(b.call(b.id("result"), MemberName(MemberSet), b.id("index"),
			b.regionCall("goRegionRead", b.thisProperty("region"), b.id("index"))))),
		b.returnStatement(b.call(b.id("result"), MemberName(MemberAppend), b.id("zero"), values)))
}

func (b pointerSliceBuilder) pointerAppendSlice() tsgo.MethodDeclaration {
	return b.method(nil, MemberName(MemberAppendSlice), nil, []tsgo.ParameterDeclaration{
		b.parameter("zero", b.typeT()), b.parameter("source", b.sliceType()),
	}, b.sliceType(),
		b.factory.VariableStatement(nil, b.factory.VariableDeclarationList([]tsgo.VariableDeclaration{
			b.factory.VariableDeclaration(b.id("values"), nil, b.factory.ArrayTypeNode(b.typeT()), b.factory.ArrayLiteralExpression(nil, false)),
		}, tsgo.NodeFlagsConst)),
		b.exactLoop(b.call(b.id("source"), MemberName(MemberSourceLength)), b.factory.ExpressionStatement(
			b.call(b.id("values"), "push", b.call(b.id("source"), MemberName(MemberGet), b.id("index"))))),
		b.returnStatement(b.call(b.factory.ThisExpression(), MemberName(MemberAppend), b.id("zero"), b.id("values"))))
}
