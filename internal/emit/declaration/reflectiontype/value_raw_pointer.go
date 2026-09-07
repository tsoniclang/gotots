package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func pointerRawOperation(context api.Context, children api.ChildEmitter, pointee types.Type) (tsgo.Expression, []api.RootRequest, error) {
	supported, err := memory.SupportsLayout(context, pointee)
	if err != nil || !supported {
		return nil, nil, err
	}
	layout, represented, err := memory.Layout(context, children, nil, pointee)
	if err != nil {
		return nil, nil, err
	}
	parameter, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return nil, nil, err
	}
	factory := context.Factory()
	storage, err := memory.ToStoragePointer(context, children, nil, pointee, api.DirectExpression(factory.Identifier(parameter)))
	if err != nil {
		return nil, nil, err
	}
	raw, err := pointermarker.Operation(context, tsoniccore.SymbolToRawPointer, []api.TypeEmission{represented}, []api.ExpressionEmission{storage, layout})
	if err != nil {
		return nil, nil, err
	}
	return factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{untypedParameter(factory, parameter)}, nil,
		factory.EqualsGreaterThanToken(), factory.Block(append(raw.Before(), factory.ReturnStatement(raw.Value())), true)), raw.Requests(), nil
}
