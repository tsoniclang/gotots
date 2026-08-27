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

func (b builder) selectSendMethod() tsgo.MethodDeclaration {
	prepared := b.id("prepared")
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
			},
			true,
		)),
	)
}
