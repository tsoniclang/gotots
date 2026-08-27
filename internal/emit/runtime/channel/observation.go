package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) observeCloseMethod() tsgo.MethodDeclaration {
	observer := b.id("observer")
	return b.method(
		nil,
		MemberObserveClose,
		[]tsgo.ParameterDeclaration{
			b.parameter("observer", b.closeObserverType()),
		},
		b.closeUnsubscribeType(),
		b.factory.IfStatement(
			b.thisProperty("closed"),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.call(observer)),
				b.returnStatement(b.arrow(
					nil,
					b.voidType(),
					b.undefined(),
				)),
			}, true),
			nil,
		),
		b.expression(b.methodCall(
			b.thisProperty("closeObservers"),
			"add",
			observer,
		)),
		b.returnStatement(b.arrow(
			nil,
			b.voidType(),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.methodCall(
					b.thisProperty("closeObservers"),
					"delete",
					observer,
				)),
			}, true),
		)),
	)
}
