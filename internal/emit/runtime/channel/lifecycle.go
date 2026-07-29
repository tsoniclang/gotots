package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) closeMethod() tsgo.MethodDeclaration {
	offer := b.variableDeclaration("offer", nil, nil)
	receiver := b.variableDeclaration("receiver", nil, nil)
	return b.method(
		nil,
		MemberClose,
		nil,
		b.voidType(),
		b.factory.IfStatement(
			b.thisProperty("closed"),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("close of closed channel")),
			}, true),
			nil,
		),
		b.expression(b.assign(
			b.thisProperty("closed"),
			b.factory.TrueLiteral(),
		)),
		b.variable(
			tsgo.NodeFlagsConst,
			"sendFailure",
			b.objectType(),
			b.panicValue("send on closed channel"),
		),
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{offer},
				tsgo.NodeFlagsConst,
			),
			b.thisProperty("senders"),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.call(
					b.element(b.id("offer"), b.number("1")),
					b.id("sendFailure"),
				)),
				b.expression(b.methodCall(
					b.thisProperty("senders"),
					"delete",
					b.id("offer"),
				)),
			}, true),
		),
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{receiver},
				tsgo.NodeFlagsConst,
			),
			b.thisProperty("receivers"),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.call(
					b.id("receiver"),
					b.receiveTuple(
						b.call(b.thisProperty("zero")),
						false,
					),
				)),
				b.expression(b.methodCall(
					b.thisProperty("receivers"),
					"delete",
					b.id("receiver"),
				)),
			}, true),
		),
	)
}
