package channel

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const MakeMember = "make"

func (b builder) channelClass() tsgo.ClassDeclaration {
	members := []tsgo.ClassElement{
		b.channelConstructor(),
		b.channelMakeMethod(),
		b.staticSendMethod(),
		b.staticReceiveMethod(),
		b.staticCloseMethod(),
		b.staticLengthMethod(),
		b.staticCapacityMethod(),
		b.staticSelectSendMethod(),
		b.staticSelectReceiveMethod(),
		b.propertyDeclaration(
			[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
			"buffer",
			b.arrayType(b.typeT()),
			b.arrayLiteral(),
		),
		b.propertyDeclaration(
			[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
			"bufferHead",
			b.numberType(),
			b.number("0"),
		),
		b.propertyDeclaration(
			[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
			"senders",
			b.typeReference("Set", b.sendOfferType()),
			b.factory.NewExpression(
				b.id("Set"),
				[]tsgo.TypeNode{b.sendOfferType()},
				nil,
			),
		),
		b.propertyDeclaration(
			[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
			"receivers",
			b.typeReference("Set", b.receiveResolverType()),
			b.factory.NewExpression(
				b.id("Set"),
				[]tsgo.TypeNode{b.receiveResolverType()},
				nil,
			),
		),
		b.propertyDeclaration(
			[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
			"closed",
			b.booleanType(),
			b.factory.FalseLiteral(),
		),
		b.sendReadyMethod(),
		b.receiveReadyMethod(),
		b.lengthMethod(),
		b.capacityMethod(),
		b.commitReceiverMethod(),
		b.commitPreparedSendMethod(),
		b.takeSenderMethod(),
		b.takeReceiveMethod(),
		b.sendMethod(),
		b.receiveMethod(),
		b.closeMethod(),
		b.commitSelectSendMethod(),
		b.subscribeSelectSendMethod(),
		b.subscribeSelectReceiveMethod(),
		b.selectSendMethod(),
		b.selectReceiveMethod(),
		b.compactBufferMethod(),
	}
	return b.factory.ClassDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		b.id(b.channelName),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.HeritageClause{
			b.factory.HeritageClause(
				tsgo.HeritageClauseTokenKindImplementsKeyword,
				[]tsgo.ExpressionWithTypeArguments{
					b.factory.ExpressionWithTypeArguments(
						b.id(b.receiveName),
						[]tsgo.TypeNode{b.typeT()},
					),
					b.factory.ExpressionWithTypeArguments(
						b.id(b.sendName),
						[]tsgo.TypeNode{b.typeT()},
					),
				},
			),
		},
		members,
	)
}

func (b builder) channelConstructor() tsgo.ConstructorDeclaration {
	return b.factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{
			b.factory.ParameterDeclaration(
				[]tsgo.ModifierLike{
					b.factory.PrivateKeyword(),
					b.factory.ReadonlyKeyword(),
				},
				nil,
				b.id("capacity"),
				nil,
				b.numberType(),
				nil,
			),
			b.factory.ParameterDeclaration(
				[]tsgo.ModifierLike{
					b.factory.PrivateKeyword(),
					b.factory.ReadonlyKeyword(),
				},
				nil,
				b.id("zero"),
				nil,
				b.zeroFunctionType(),
				nil,
			),
			b.factory.ParameterDeclaration(
				[]tsgo.ModifierLike{
					b.factory.PrivateKeyword(),
					b.factory.ReadonlyKeyword(),
				},
				nil,
				b.id("copy"),
				nil,
				b.copyFunctionType(),
				nil,
			),
		},
		nil,
		b.factory.Block(nil, true),
	)
}

func (b builder) channelMakeMethod() tsgo.MethodDeclaration {
	typeT := b.typeT()
	numericCapacity := b.id("numericCapacity")
	return b.factory.MethodDeclaration(
		[]tsgo.ModifierLike{
			b.factory.StaticKeyword(),
		},
		nil,
		b.id(MakeMember),
		nil,
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		[]tsgo.ParameterDeclaration{
			b.parameter("capacity", b.integerInputType()),
			b.parameter("zero", b.zeroFunctionType()),
			b.parameter("copy", b.copyFunctionType()),
		},
		b.typeReference(b.channelName, typeT),
		b.factory.Block(
			[]tsgo.Statement{
				b.variable(
					tsgo.NodeFlagsConst,
					"numericCapacity",
					b.numberType(),
					b.toNumber(b.id("capacity")),
				),
				b.factory.IfStatement(
					b.logicalOr(
						b.logicalNot(b.staticCall(
							"Number",
							"isSafeInteger",
							numericCapacity,
						)),
						b.binary(
							numericCapacity,
							tsgo.BinaryOperatorLessThanToken,
							b.number("0"),
						),
					),
					b.factory.Block(
						[]tsgo.Statement{
							b.expression(b.panic(
								"makechan: size out of range",
							)),
						},
						true,
					),
					nil,
				),
				b.returnStatement(
					b.factory.NewExpression(
						b.id(b.channelName),
						[]tsgo.TypeNode{typeT},
						[]tsgo.Expression{
							numericCapacity,
							b.id("zero"),
							b.id("copy"),
						},
					),
				),
			},
			true,
		),
	)
}

func (b builder) zeroFunctionType() tsgo.TypeNode {
	return b.functionType(nil, b.typeT())
}

func (b builder) copyFunctionType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{b.parameter("value", b.typeT())},
		b.typeT(),
	)
}

func (b builder) lengthMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		channelLengthMember,
		nil,
		b.numberType(),
		b.returnStatement(b.liveLength("buffer", "bufferHead")),
	)
}

func (b builder) capacityMethod() tsgo.MethodDeclaration {
	return b.method(
		nil,
		channelCapacityMember,
		nil,
		b.numberType(),
		b.returnStatement(b.thisProperty("capacity")),
	)
}

func (b builder) sendResolverType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"failure",
				b.unionType(b.objectType(), b.undefinedType()),
			),
		},
		b.voidType(),
	)
}

func (b builder) receiveResultType() tsgo.TypeNode {
	return b.tupleType(b.typeT(), b.booleanType())
}

func (b builder) receiveResolverType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter("result", b.receiveResultType()),
		},
		b.booleanType(),
	)
}

func (b builder) acceptType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
			b.parameter("ok", b.booleanType()),
		},
		b.voidType(),
	)
}

func (b builder) selectClaimType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter(
				"failure",
				b.unionType(b.objectType(), b.undefinedType()),
			),
		},
		b.booleanType(),
	)
}

func (b builder) selectCommitType() tsgo.TypeNode {
	return b.unionType(b.booleanType(), b.objectType())
}

func (b builder) selectSendResultType() tsgo.TypeNode {
	return b.unionType(
		b.tupleType(b.typeT()),
		b.undefinedType(),
	)
}

func (b builder) sendTakeType() tsgo.TypeNode {
	return b.functionType(nil, b.selectSendResultType())
}

func (b builder) sendFailureType() tsgo.TypeNode {
	return b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter("failure", b.objectType()),
		},
		b.booleanType(),
	)
}

func (b builder) sendOfferType() tsgo.TypeNode {
	return b.tupleType(
		b.sendTakeType(),
		b.sendFailureType(),
	)
}
