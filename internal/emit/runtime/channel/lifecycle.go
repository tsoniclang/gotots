package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) closeMethod() tsgo.MethodDeclaration {
	observer := b.variableDeclaration("observer", nil, nil)
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
		b.factory.ForOfStatement(
			nil,
			b.factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{observer},
				tsgo.NodeFlagsConst,
			),
			b.thisProperty("closeObservers"),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.call(b.id("observer"))),
			}, true),
		),
		b.expression(b.methodCall(
			b.thisProperty("closeObservers"),
			"clear",
		)),
	)
}
