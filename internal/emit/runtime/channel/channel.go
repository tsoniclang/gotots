package channel

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
			"closed",
			b.booleanType(),
			b.factory.FalseLiteral(),
		),
		b.propertyDeclaration(
			[]tsgo.ModifierLike{b.factory.PrivateKeyword()},
			"closeObservers",
			b.typeReference("Set", b.closeObserverType()),
			b.factory.NewExpression(
				b.id("Set"),
				[]tsgo.TypeNode{b.closeObserverType()},
				nil,
			),
		),
		b.sendReadyMethod(),
		b.receiveReadyMethod(),
		b.lengthMethod(),
		b.capacityMethod(),
		b.commitPreparedSendMethod(),
		b.takeReceiveMethod(),
		b.sendMethod(),
		b.receiveMethod(),
		b.closeMethod(),
		b.observeCloseMethod(),
		b.commitSelectSendMethod(),
		b.selectSendMethod(),
		b.selectReceiveMethod(),
		b.compactBufferMethod(),
	}
	return typescriptclass.Declaration(b.factory,
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
						b.logicalNot(b.methodCall(
							api.TargetIntrinsicNumber.Expression(b.factory),
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

func (b builder) receiveResultType() tsgo.TypeNode {
	return b.tupleType(b.typeT(), b.booleanType())
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

func (b builder) closeObserverType() tsgo.TypeNode {
	return b.functionType(nil, b.voidType())
}

func (b builder) closeUnsubscribeType() tsgo.TypeNode {
	return b.functionType(nil, b.voidType())
}

func (b builder) selectCommitType() tsgo.TypeNode {
	return b.unionType(b.booleanType(), b.objectType())
}
