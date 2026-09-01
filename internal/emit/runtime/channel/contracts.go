package channel

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	MemberSend            = "send"
	MemberReceive         = "receive"
	MemberClose           = "close"
	MemberLength          = "length"
	MemberCapacity        = "capacity"
	MemberSelectSend      = "$selectSend"
	MemberSelectReceive   = "$selectReceive"
	MemberObserveClose    = "$observeClose"
	channelLengthMember   = "$length"
	channelCapacityMember = "$capacity"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	channelName string,
	receiveName string,
	sendName string,
	caseName string,
	selectName string,
	spawnName string,
	selectReadyName string,
	selectAttemptName string,
	panicName string,
) (tsgo.Statement, error) {
	target := builder{
		factory:           factory,
		channelName:       channelName,
		receiveName:       receiveName,
		sendName:          sendName,
		caseName:          caseName,
		selectName:        selectName,
		spawnName:         spawnName,
		selectReadyName:   selectReadyName,
		selectAttemptName: selectAttemptName,
		panicName:         panicName,
	}
	switch symbol {
	case api.RuntimeChannel:
		return target.channelClass(), nil
	case api.RuntimeReceiveChannel:
		return target.receiveContract(), nil
	case api.RuntimeSendChannel:
		return target.sendContract(), nil
	case api.RuntimeSelectCase:
		return target.selectCaseContract(), nil
	case api.RuntimeSelect:
		return target.selectFunction(), nil
	case api.RuntimeGoSpawn:
		return target.spawnFunction(), nil
	case api.RuntimeSelectReady:
		return target.selectReadyFunction(), nil
	case api.RuntimeSelectAttempt:
		return target.selectAttemptFunction(), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func (b builder) selectCaseContract() tsgo.InterfaceDeclaration {
	return b.factory.InterfaceDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		b.id(b.caseName),
		nil,
		nil,
		[]tsgo.TypeElement{
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id("ready"),
				nil,
				nil,
				nil,
				b.booleanType(),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id("commit"),
				nil,
				nil,
				nil,
				b.selectCommitType(),
			),
		},
	)
}

func (b builder) receiveContract() tsgo.InterfaceDeclaration {
	result := b.tupleType(b.typeT(), b.booleanType())
	accept := b.functionType(
		[]tsgo.ParameterDeclaration{
			b.parameter("value", b.typeT()),
			b.parameter("ok", b.booleanType()),
		},
		b.voidType(),
	)
	return b.factory.InterfaceDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		b.id(b.receiveName),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		nil,
		[]tsgo.TypeElement{
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(channelLengthMember),
				nil,
				nil,
				nil,
				b.numberType(),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(channelCapacityMember),
				nil,
				nil,
				nil,
				b.numberType(),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(MemberReceive),
				nil,
				nil,
				nil,
				b.blockingResultType(result),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(MemberSelectReceive),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("accept", accept),
				},
				b.typeReference(b.caseName),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(MemberObserveClose),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("observer", b.closeObserverType()),
				},
				b.closeUnsubscribeType(),
			),
		},
	)
}

func (b builder) sendContract() tsgo.InterfaceDeclaration {
	return b.factory.InterfaceDeclaration(
		[]tsgo.ModifierLike{b.factory.ExportKeyword()},
		b.id(b.sendName),
		[]tsgo.TypeParameterDeclaration{b.typeParameter()},
		nil,
		[]tsgo.TypeElement{
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(channelLengthMember),
				nil,
				nil,
				nil,
				b.numberType(),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(channelCapacityMember),
				nil,
				nil,
				nil,
				b.numberType(),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(MemberSend),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("value", b.typeT()),
				},
				b.blockingResultType(b.voidType()),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(MemberClose),
				nil,
				nil,
				nil,
				b.voidType(),
			),
			b.factory.MethodSignatureDeclaration(
				nil,
				b.id(MemberSelectSend),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					b.parameter("value", b.typeT()),
				},
				b.typeReference(b.caseName),
			),
		},
	)
}
