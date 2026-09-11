package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func arrayStoragePointer(context api.Context, children api.ChildEmitter, source ast.Node, pointee types.Type, pointer api.ExpressionEmission) (api.ExpressionEmission, error) {
	model, ok := arrayvalue.Resolve(context, pointee)
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	logical, err := children.RepresentedType(context, source, pointee)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	inputType, err := pointermarker.Type(context, logical, true)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayLayout, storage, err := Layout(context, children, source, pointee)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	outputType, err := pointermarker.Type(context, storage, true)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	childLayout, element, err := Layout(context, children, source, model.ElementType())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storedElement, err := context.ContainerStorage().ContainerStorageType(context, source, model.ElementType())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parameterName, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	locationName, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	parameter := factory.Identifier(parameterName)
	loaded, err := pointermarker.Operation(context, tsoniccore.SymbolLoadPointer, []api.TypeEmission{logical}, []api.ExpressionEmission{api.DirectExpression(parameter)})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	location, err := model.Location(context, children, loaded)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	address, err := context.Names().Runtime(api.RuntimeRegionAddress, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	base := api.DirectExpression(factory.CallExpression(address.Expression(factory), nil,
		[]tsgo.TypeNode{storedElement.Value()}, []tsgo.Expression{factory.Identifier(locationName), factory.NumericLiteral("0", tsgo.TokenFlagsNone)},
		tsgo.NodeFlagsNone), api.CombineRequests(storedElement.Requests(), address.Requests())...)
	base, err = ElementToMemoryPointer(context, children, source, model.ElementType(), base)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	raw, err := pointermarker.Operation(context, tsoniccore.SymbolToRawPointer, []api.TypeEmission{element}, []api.ExpressionEmission{base, childLayout})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result, err := pointermarker.Operation(context, tsoniccore.SymbolReinterpretRawPointer, []api.TypeEmission{storage}, []api.ExpressionEmission{raw, arrayLayout})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	variable := func(name string, value tsgo.Expression) tsgo.Statement {
		return factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
			factory.VariableDeclaration(factory.Identifier(name), nil, nil, value),
		}, tsgo.NodeFlagsConst))
	}
	undefined := factory.VoidExpression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))
	body := []tsgo.Statement{factory.IfStatement(factory.BinaryExpression(nil, parameter, nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken), undefined),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(undefined)}, true), nil)}
	body = append(body, location.Before()...)
	body = append(body, variable(locationName, location.Value()))
	body = append(body, result.Before()...)
	body = append(body, factory.ReturnStatement(result.Value()))
	transform := factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
		factory.ParameterDeclaration(nil, nil, parameter, nil, inputType.Value(), nil),
	}, outputType.Value(), factory.EqualsGreaterThanToken(), factory.Block(body, true))
	return api.NewExpressionEmission(pointer.Before(), factory.CallExpression(factory.ParenthesizedExpression(transform), nil, nil,
		[]tsgo.Expression{pointer.Value()}, tsgo.NodeFlagsNone), api.CombineRequests(pointer.Requests(), inputType.Requests(), outputType.Requests(), location.Requests(), result.Requests()))
}
