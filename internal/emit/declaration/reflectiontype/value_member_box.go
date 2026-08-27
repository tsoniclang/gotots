package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type reflectionMemberBox struct {
	sourceType     types.Type
	adapter        api.NameReference
	interfaceValue bool
}

func newReflectionMemberBox(
	context api.Context,
	sourceType types.Type,
) (reflectionMemberBox, error) {
	_, interfaceValue := types.Unalias(sourceType).Underlying().(*types.Interface)
	if interfaceValue {
		return reflectionMemberBox{
			sourceType:     sourceType,
			interfaceValue: true,
		}, nil
	}
	adapter, err := context.Names().InterfaceAdapter(sourceType, nil)
	if err != nil {
		return reflectionMemberBox{}, err
	}
	return reflectionMemberBox{
		sourceType: sourceType,
		adapter:    adapter,
	}, nil
}

func (m reflectionMemberBox) requests() []api.RootRequest {
	if m.interfaceValue {
		return nil
	}
	return m.adapter.Requests()
}

func (m reflectionMemberBox) boxedValue(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.Expression {
	if m.interfaceValue {
		return value
	}
	return factory.NewExpression(
		m.adapter.Expression(factory),
		nil,
		[]tsgo.Expression{value},
	)
}

func (m reflectionMemberBox) admittedValue(
	context api.Context,
	operand string,
	operation string,
	scaffold *locationScaffold,
) (api.ExpressionEmission, error) {
	value := scaffold.factory.Identifier(operand)
	if !m.interfaceValue {
		return api.DirectExpression(guardedForeignOperand(
			scaffold,
			m.adapter,
			operand,
			operation,
		)), nil
	}
	admitted, requests, err := admittedInterfaceValue(
		context,
		m.sourceType,
		value,
		operation,
		scaffold,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(admitted, requests...), nil
}
