package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ProjectElementPointer(context api.Context, source ast.Node, element types.Type, pointer api.ExpressionEmission) (api.ExpressionEmission, error) {
	required, err := RequiresProjection(context, element)
	if err != nil || !required {
		return pointer, err
	}
	physical, err := context.Values().MemoryStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := context.ContainerStorage().ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	physicalName, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storedName, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	logical, err := context.Values().FromMemoryStorage(context, source, element, api.DirectExpression(factory.Identifier(physicalName)))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	read, err := context.ContainerStorage().ToContainerStorage(context, source, element, logical)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logical, err = context.ContainerStorage().FromContainerStorage(context, source, element, api.DirectExpression(factory.Identifier(storedName)))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	write, err := context.Values().ToMemoryStorage(context, source, element, logical)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	conversion := func(name string, input, output api.TypeEmission, body api.ExpressionEmission) api.ExpressionEmission {
		return api.DirectExpression(factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(nil, nil, factory.Identifier(name), nil, input.Value(), nil),
		}, output.Value(), factory.EqualsGreaterThanToken(), factory.Block(append(body.Before(), factory.ReturnStatement(body.Value())), true)),
			api.CombineRequests(input.Requests(), output.Requests(), body.Requests())...)
	}
	return pointermarker.Operation(context, tsoniccore.SymbolProjectPointer, []api.TypeEmission{physical, stored}, []api.ExpressionEmission{
		pointer, conversion(physicalName, physical, stored, read), conversion(storedName, stored, physical, write),
	})
}

func ElementToMemoryPointer(context api.Context, children api.ChildEmitter, source ast.Node, element types.Type, pointer api.ExpressionEmission) (api.ExpressionEmission, error) {
	logical, err := context.Values().ProjectStoragePointer(context, source, element, pointer)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return ToStoragePointer(context, children, source, element, logical)
}
