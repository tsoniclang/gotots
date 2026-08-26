package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) commitPreparedSendMethod() tsgo.MethodDeclaration {
	buffer := b.thisProperty("buffer")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"commitPreparedSend",
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
		},
		b.booleanType(),
		b.factory.IfStatement(
			b.thisProperty("closed"),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("send on closed channel")),
			}, true),
			nil,
		),
		b.factory.IfStatement(
			b.binary(
				b.liveLength("buffer", "bufferHead"),
				tsgo.BinaryOperatorLessThanToken,
				b.thisProperty("capacity"),
			),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.methodCall(
					buffer,
					"push",
					b.id("value"),
				)),
				b.returnStatement(b.factory.TrueLiteral()),
			}, true),
			nil,
		),
		b.returnStatement(b.factory.FalseLiteral()),
	)
}

func (b builder) takeReceiveMethod() tsgo.MethodDeclaration {
	resultType := b.unionType(
		b.receiveResultType(),
		b.undefinedType(),
	)
	buffer := b.thisProperty("buffer")
	bufferHead := b.thisProperty("bufferHead")
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"takeReceive",
		nil,
		resultType,
		b.factory.IfStatement(
			b.binary(
				b.liveLength("buffer", "bufferHead"),
				tsgo.BinaryOperatorGreaterThanToken,
				b.number("0"),
			),
			b.factory.Block([]tsgo.Statement{
				b.variable(
					tsgo.NodeFlagsConst,
					"value",
					b.typeT(),
					b.denseElement(buffer, bufferHead, b.typeT()),
				),
				b.expression(b.increment(bufferHead)),
				b.expression(b.methodCall(
					b.factory.ThisExpression(),
					"compactBuffer",
				)),
				b.returnStatement(b.receiveTuple(
					b.id("value"),
					true,
				)),
			}, true),
			nil,
		),
		b.factory.IfStatement(
			b.thisProperty("closed"),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.receiveTuple(
					b.call(b.thisProperty("zero")),
					false,
				)),
			}, true),
			nil,
		),
		b.returnStatement(b.undefined()),
	)
}

func (b builder) sendMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		MemberSend,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
		},
		b.voidType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"prepared",
			b.typeT(),
			b.copyCall(b.id("value")),
		),
		b.factory.IfStatement(
			b.logicalNot(b.methodCall(
				b.factory.ThisExpression(),
				"commitPreparedSend",
				b.id("prepared"),
			)),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("serial channel send would block")),
			}, true),
			nil,
		),
	)
}

func (b builder) receiveMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		MemberReceive,
		nil,
		b.receiveResultType(),
		b.variable(
			tsgo.NodeFlagsConst,
			"immediate",
			b.unionType(
				b.receiveResultType(),
				b.undefinedType(),
			),
			b.methodCall(
				b.factory.ThisExpression(),
				"takeReceive",
			),
		),
		b.factory.IfStatement(
			b.strictUndefined(b.id("immediate")),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("serial channel receive would block")),
			}, true),
			nil,
		),
		b.returnStatement(b.id("immediate")),
	)
}

func (b builder) copyCall(value tsgo.Expression) tsgo.CallExpression {
	return b.call(b.thisProperty("copy"), value)
}

func (b builder) receiveTuple(
	value tsgo.Expression,
	ok bool,
) tsgo.ArrayLiteralExpression {
	status := tsgo.Expression(b.factory.FalseLiteral())
	if ok {
		status = b.factory.TrueLiteral()
	}
	return b.receiveTupleExpression(value, status)
}

func (b builder) receiveTupleExpression(
	value tsgo.Expression,
	ok tsgo.Expression,
) tsgo.ArrayLiteralExpression {
	return b.arrayLiteral(value, ok)
}
