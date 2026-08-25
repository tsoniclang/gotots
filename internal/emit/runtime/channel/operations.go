package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) commitReceiverMethod() tsgo.MethodDeclaration {
	receiver := b.variableDeclaration("receiver", nil, nil)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"commitReceiver",
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
			b.parameter("ok", b.booleanType()),
		},
		b.booleanType(),
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{receiver},
				tsgo.NodeFlagsConst,
			),
			b.thisProperty("receivers"),
			b.factory.Block([]tsgo.Statement{
				b.variable(
					tsgo.NodeFlagsConst,
					"accepted",
					b.booleanType(),
					b.call(
						b.id("receiver"),
						b.receiveTupleExpression(
							b.id("value"),
							b.id("ok"),
						),
					),
				),
				b.expression(b.methodCall(
					b.thisProperty("receivers"),
					"delete",
					b.id("receiver"),
				)),
				b.factory.IfStatement(
					b.id("accepted"),
					b.factory.Block([]tsgo.Statement{
						b.returnStatement(b.factory.TrueLiteral()),
					}, true),
					nil,
				),
			}, true),
		),
		b.returnStatement(b.factory.FalseLiteral()),
	)
}

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
			b.methodCall(
				b.factory.ThisExpression(),
				"commitReceiver",
				b.id("value"),
				b.factory.TrueLiteral(),
			),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.factory.TrueLiteral()),
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

func (b builder) takeSenderMethod() tsgo.MethodDeclaration {
	offer := b.variableDeclaration("offer", nil, nil)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"takeSender",
		nil,
		b.selectSendResultType(),
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{offer},
				tsgo.NodeFlagsConst,
			),
			b.thisProperty("senders"),
			b.factory.Block([]tsgo.Statement{
				b.variable(
					tsgo.NodeFlagsConst,
					"selected",
					b.selectSendResultType(),
					b.call(b.element(
						b.id("offer"),
						b.number("0"),
					)),
				),
				b.expression(b.methodCall(
					b.thisProperty("senders"),
					"delete",
					b.id("offer"),
				)),
				b.factory.IfStatement(
					b.strictDefined(b.id("selected")),
					b.factory.Block([]tsgo.Statement{
						b.returnStatement(b.id("selected")),
					}, true),
					nil,
				),
			}, true),
		),
		b.returnStatement(b.undefined()),
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
				b.variable(
					tsgo.NodeFlagsConst,
					"waiting",
					b.selectSendResultType(),
					b.methodCall(
						b.factory.ThisExpression(),
						"takeSender",
					),
				),
				b.factory.IfStatement(
					b.strictDefined(b.id("waiting")),
					b.factory.Block([]tsgo.Statement{
						b.expression(b.methodCall(
							buffer,
							"push",
							b.element(
								b.id("waiting"),
								b.number("0"),
							),
						)),
					}, true),
					nil,
				),
				b.returnStatement(b.receiveTuple(
					b.id("value"),
					true,
				)),
			}, true),
			nil,
		),
		b.variable(
			tsgo.NodeFlagsConst,
			"selected",
			b.selectSendResultType(),
			b.methodCall(
				b.factory.ThisExpression(),
				"takeSender",
			),
		),
		b.factory.IfStatement(
			b.strictDefined(b.id("selected")),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.receiveTuple(
					b.element(b.id("selected"), b.number("0")),
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
	if !b.cooperative() {
		return b.synchronousSendMethod()
	}
	return b.method(
		nil,
		MemberSend,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
		},
		b.promiseType(b.voidType()),
		b.variable(
			tsgo.NodeFlagsConst,
			"prepared",
			b.typeT(),
			b.copyCall(b.id("value")),
		),
		b.factory.IfStatement(
			b.methodCall(
				b.factory.ThisExpression(),
				"commitPreparedSend",
				b.id("prepared"),
			),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.promiseResolve(nil)),
			}, true),
			nil,
		),
		b.returnStatement(b.newPromise(
			b.voidType(),
			b.sendExecutor(),
		)),
	)
}

func (b builder) synchronousSendMethod() tsgo.MethodDeclaration {
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
				b.expression(b.panic("synchronous channel send would block")),
			}, true),
			nil,
		),
	)
}

func (b builder) sendExecutor() tsgo.ArrowFunction {
	failureType := b.unionType(b.objectType(), b.undefinedType())
	finish := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", failureType),
		},
		b.voidType(),
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.strictUndefined(b.id("failure")),
				b.factory.Block([]tsgo.Statement{
					b.expression(b.call(b.id("resolve"))),
				}, true),
				b.factory.Block([]tsgo.Statement{
					b.expression(b.call(
						b.id("reject"),
						b.id("failure"),
					)),
				}, true),
			),
		}, true),
	)
	take := b.arrow(
		nil,
		b.selectSendResultType(),
		b.factory.Block([]tsgo.Statement{
			b.expression(b.call(b.id("finish"), b.undefined())),
			b.returnStatement(b.arrayLiteral(b.id("prepared"))),
		}, true),
	)
	fail := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.expression(b.call(b.id("finish"), b.id("failure"))),
			b.returnStatement(b.factory.TrueLiteral()),
		}, true),
	)
	return b.arrow(
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				nil,
				nil,
				b.id("resolve"),
				nil,
				nil,
				nil,
			),
			b.factory.ParameterDeclaration(
				nil,
				nil,
				b.id("reject"),
				nil,
				nil,
				nil,
			),
		},
		b.voidType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"finish",
				b.sendResolverType(),
				finish,
			),
			b.variable(
				tsgo.NodeFlagsConst,
				"offer",
				b.sendOfferType(),
				b.arrayLiteral(take, fail),
			),
			b.expression(b.methodCall(
				b.thisProperty("senders"),
				"add",
				b.id("offer"),
			)),
		}, true),
	)
}

func (b builder) receiveMethod() tsgo.MethodDeclaration {
	if !b.cooperative() {
		return b.synchronousReceiveMethod()
	}
	receive := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("result", b.receiveResultType()),
		},
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.expression(b.call(b.id("resolve"), b.id("result"))),
			b.returnStatement(b.factory.TrueLiteral()),
		}, true),
	)
	return b.method(
		nil,
		MemberReceive,
		nil,
		b.promiseType(b.receiveResultType()),
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
			b.strictDefined(b.id("immediate")),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.promiseResolve(b.id("immediate"))),
			}, true),
			nil,
		),
		b.returnStatement(b.newPromise(
			b.receiveResultType(),
			b.arrow(
				[]tsgo.ParameterDeclaration{
					b.factory.ParameterDeclaration(
						nil,
						nil,
						b.id("resolve"),
						nil,
						nil,
						nil,
					),
				},
				b.voidType(),
				b.factory.Block([]tsgo.Statement{
					b.variable(
						tsgo.NodeFlagsConst,
						"receive",
						b.receiveResolverType(),
						receive,
					),
					b.expression(b.methodCall(
						b.thisProperty("receivers"),
						"add",
						b.id("receive"),
					)),
				}, true),
			),
		)),
	)
}

func (b builder) synchronousReceiveMethod() tsgo.MethodDeclaration {
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
				b.expression(b.panic("synchronous channel receive would block")),
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
