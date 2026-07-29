package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) commitSelectSendMethod() tsgo.MethodDeclaration {
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"commitSelectSend",
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
		},
		b.selectCommitType(),
		b.factory.IfStatement(
			b.thisProperty("closed"),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.panicValue("send on closed channel")),
			}, true),
			nil,
		),
		b.returnStatement(b.methodCall(
			b.factory.ThisExpression(),
			"commitPreparedSend",
			b.id("value"),
		)),
	)
}

func (b builder) subscribeSelectSendMethod() tsgo.MethodDeclaration {
	take := b.arrow(
		nil,
		b.selectSendResultType(),
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.logicalNot(b.call(b.id("claim"), b.undefined())),
				b.factory.Block([]tsgo.Statement{
					b.returnStatement(b.undefined()),
				}, true),
				nil,
			),
			b.returnStatement(b.arrayLiteral(b.id("value"))),
		}, true),
	)
	fail := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.returnStatement(b.call(
				b.id("claim"),
				b.id("failure"),
			)),
		}, true),
	)
	cancel := b.arrow(
		nil,
		b.voidType(),
		b.factory.Block([]tsgo.Statement{
			b.expression(b.methodCall(
				b.thisProperty("senders"),
				"delete",
				b.id("offer"),
			)),
		}, true),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"subscribeSelectSend",
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
			b.parameter("claim", b.selectClaimType()),
		},
		b.functionType(nil, b.voidType()),
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
		b.returnStatement(cancel),
	)
}

func (b builder) subscribeSelectReceiveMethod() tsgo.MethodDeclaration {
	receive := b.arrow(
		[]tsgo.ParameterDeclaration{
			b.parameter("result", b.receiveResultType()),
		},
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.factory.IfStatement(
				b.logicalNot(b.call(b.id("claim"), b.undefined())),
				b.factory.Block([]tsgo.Statement{
					b.returnStatement(b.factory.FalseLiteral()),
				}, true),
				nil,
			),
			b.expression(b.call(
				b.id("accept"),
				b.element(b.id("result"), b.number("0")),
				b.element(b.id("result"), b.number("1")),
			)),
			b.returnStatement(b.factory.TrueLiteral()),
		}, true),
	)
	cancel := b.arrow(
		nil,
		b.voidType(),
		b.factory.Block([]tsgo.Statement{
			b.expression(b.methodCall(
				b.thisProperty("receivers"),
				"delete",
				b.id("receive"),
			)),
		}, true),
	)
	return b.method(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		"subscribeSelectReceive",
		[]tsgo.ParameterDeclaration{
			b.parameter("accept", b.acceptType()),
			b.parameter("claim", b.selectClaimType()),
		},
		b.functionType(nil, b.voidType()),
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
		b.returnStatement(cancel),
	)
}

func (b builder) selectSendMethod() tsgo.MethodDeclaration {
	prepared := b.id("prepared")
	cancelType := b.functionType(nil, b.voidType())
	return b.method(
		nil,
		MemberSelectSend,
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
		},
		b.typeReference(b.caseName),
		b.variable(
			tsgo.NodeFlagsConst,
			"prepared",
			b.typeT(),
			b.copyCall(b.id("value")),
		),
		b.returnStatement(b.factory.ObjectLiteralExpression(
			[]tsgo.ObjectLiteralElementLike{
				b.propertyFunction(
					"ready",
					nil,
					b.booleanType(),
					b.methodCall(
						b.factory.ThisExpression(),
						"sendReady",
					),
				),
				b.propertyFunction(
					"commit",
					nil,
					b.selectCommitType(),
					b.methodCall(
						b.factory.ThisExpression(),
						"commitSelectSend",
						prepared,
					),
				),
				b.propertyFunction(
					"subscribe",
					[]tsgo.ParameterDeclaration{
						b.parameter("claim", b.selectClaimType()),
					},
					cancelType,
					b.methodCall(
						b.factory.ThisExpression(),
						"subscribeSelectSend",
						prepared,
						b.id("claim"),
					),
				),
			},
			true,
		)),
	)
}

func (b builder) selectReceiveMethod() tsgo.MethodDeclaration {
	commit := b.arrow(
		nil,
		b.booleanType(),
		b.factory.Block([]tsgo.Statement{
			b.variable(
				tsgo.NodeFlagsConst,
				"result",
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
				b.strictUndefined(b.id("result")),
				b.factory.Block([]tsgo.Statement{
					b.returnStatement(b.factory.FalseLiteral()),
				}, true),
				nil,
			),
			b.expression(b.call(
				b.id("accept"),
				b.element(b.id("result"), b.number("0")),
				b.element(b.id("result"), b.number("1")),
			)),
			b.returnStatement(b.factory.TrueLiteral()),
		}, true),
	)
	cancelType := b.functionType(nil, b.voidType())
	return b.method(
		nil,
		MemberSelectReceive,
		[]tsgo.ParameterDeclaration{
			b.parameter("accept", b.acceptType()),
		},
		b.typeReference(b.caseName),
		b.returnStatement(b.factory.ObjectLiteralExpression(
			[]tsgo.ObjectLiteralElementLike{
				b.propertyFunction(
					"ready",
					nil,
					b.booleanType(),
					b.methodCall(
						b.factory.ThisExpression(),
						"receiveReady",
					),
				),
				b.factory.PropertyAssignment(
					nil,
					b.id("commit"),
					nil,
					b.functionType(nil, b.selectCommitType()),
					commit,
				),
				b.propertyFunction(
					"subscribe",
					[]tsgo.ParameterDeclaration{
						b.parameter("claim", b.selectClaimType()),
					},
					cancelType,
					b.methodCall(
						b.factory.ThisExpression(),
						"subscribeSelectReceive",
						b.id("accept"),
						b.id("claim"),
					),
				),
			},
			true,
		)),
	)
}
