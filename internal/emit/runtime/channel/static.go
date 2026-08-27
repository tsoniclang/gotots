package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func (b builder) staticSendMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	return b.staticGenericMethod(
		MemberSend,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"channel",
				b.unionType(
					b.typeReference(b.sendName, typeT),
					b.undefinedType(),
				),
			),
			b.parameter("value", typeT),
		},
		b.voidType(),
		b.factory.IfStatement(
			b.strictUndefined(b.id("channel")),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("serial channel send would block")),
			}, true),
			nil,
		),
		b.expression(b.methodCall(
			b.id("channel"),
			MemberSend,
			b.id("value"),
		)),
	)
}

func (b builder) staticReceiveMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	result := b.receiveResultType()
	return b.staticGenericMethod(
		MemberReceive,
		[]tsgo.ParameterDeclaration{b.parameter(
			"channel",
			b.unionType(
				b.typeReference(b.receiveName, typeT),
				b.undefinedType(),
			),
		)},
		result,
		b.factory.IfStatement(
			b.strictUndefined(b.id("channel")),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("serial channel receive would block")),
			}, true),
			nil,
		),
		b.returnStatement(b.methodCall(
			b.id("channel"),
			MemberReceive,
		)),
	)
}

func (b builder) staticCloseMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	return b.staticGenericMethod(
		MemberClose,
		[]tsgo.ParameterDeclaration{b.parameter(
			"channel",
			b.unionType(
				b.typeReference(b.sendName, typeT),
				b.undefinedType(),
			),
		)},
		b.voidType(),
		b.factory.IfStatement(
			b.strictUndefined(b.id("channel")),
			b.factory.Block([]tsgo.Statement{
				b.expression(b.panic("close of nil channel")),
			}, true),
			nil,
		),
		b.expression(b.methodCall(b.id("channel"), MemberClose)),
	)
}

func (b builder) staticLengthMethod() tsgo.MethodDeclaration {
	return b.staticMeasureMethod(MemberLength, channelLengthMember)
}

func (b builder) staticCapacityMethod() tsgo.MethodDeclaration {
	return b.staticMeasureMethod(MemberCapacity, channelCapacityMember)
}

func (b builder) staticMeasureMethod(
	name string,
	member string,
) tsgo.MethodDeclaration {
	typeT := b.typeT()
	channelType := b.unionType(
		b.typeReference(b.receiveName, typeT),
		b.typeReference(b.sendName, typeT),
		b.undefinedType(),
	)
	return b.staticGenericMethod(
		name,
		[]tsgo.ParameterDeclaration{
			b.parameter("channel", channelType),
		},
		b.numberType(),
		b.factory.IfStatement(
			b.strictUndefined(b.id("channel")),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.number("0")),
			}, true),
			nil,
		),
		b.returnStatement(b.methodCall(b.id("channel"), member)),
	)
}

func (b builder) staticSelectSendMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	return b.staticGenericMethod(
		MemberSelectSend,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"channel",
				b.unionType(
					b.typeReference(b.sendName, typeT),
					b.undefinedType(),
				),
			),
			b.parameter("value", typeT),
		},
		b.typeReference(b.caseName),
		b.factory.IfStatement(
			b.strictUndefined(b.id("channel")),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.inertSelectCase()),
			}, true),
			nil,
		),
		b.returnStatement(b.methodCall(
			b.id("channel"),
			MemberSelectSend,
			b.id("value"),
		)),
	)
}

func (b builder) staticSelectReceiveMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	return b.staticGenericMethod(
		MemberSelectReceive,
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"channel",
				b.unionType(
					b.typeReference(b.receiveName, typeT),
					b.undefinedType(),
				),
			),
			b.parameter("accept", b.acceptType()),
		},
		b.typeReference(b.caseName),
		b.factory.IfStatement(
			b.strictUndefined(b.id("channel")),
			b.factory.Block([]tsgo.Statement{
				b.returnStatement(b.inertSelectCase()),
			}, true),
			nil,
		),
		b.returnStatement(b.methodCall(
			b.id("channel"),
			MemberSelectReceive,
			b.id("accept"),
		)),
	)
}

func (b builder) staticGenericMethod(
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements ...tsgo.Statement,
) tsgo.MethodDeclaration {
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{b.factory.StaticKeyword()},
		nil,
		b.id(name),
		nil,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		parameters,
		result,
		b.factory.Block(statements, true),
	)
}

func (b builder) inertSelectCase() tsgo.ObjectLiteralExpression {
	return b.factory.ObjectLiteralExpression(
		[]tsgo.ObjectLiteralElementLike{
			b.propertyFunction(
				"ready",
				nil,
				b.booleanType(),
				b.factory.FalseLiteral(),
			),
			b.propertyFunction(
				"commit",
				nil,
				b.booleanType(),
				b.factory.FalseLiteral(),
			),
		},
		true,
	)
}
