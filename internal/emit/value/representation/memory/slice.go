package memory

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	descriptor "github.com/tsoniclang/gotots/internal/emit/runtime/memorydescriptor"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	sliceruntime "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	descriptorvalue "github.com/tsoniclang/gotots/internal/emit/value/memorydescriptor"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) sliceToDescriptor(context api.Context, source ast.Node, model descriptorvalue.Model, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	slice, element, ok := slicevalue.Source(model.SourceType())
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	projected, err := slicevalue.Project(context, model.SourceType(), value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logical, err := owner.children.RepresentedType(context, source, slice)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := model.StorageType(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	parameter := factory.Identifier(name)
	elementStorage, err := owner.ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := owner.ContainerStorageZero(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	dataOperation, err := context.Names().Runtime(api.RuntimeSliceData, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zeroFactory := factory.ArrowFunction(nil, nil, nil, elementStorage.Value(), factory.EqualsGreaterThanToken(),
		factory.Block(append(zero.Before(), factory.ReturnStatement(zero.Value())), true))
	elementPointer := api.DirectExpression(factory.CallExpression(dataOperation.Expression(factory), nil, []tsgo.TypeNode{elementStorage.Value()},
		[]tsgo.Expression{parameter, zeroFactory}, tsgo.NodeFlagsNone), api.CombineRequests(dataOperation.Requests(), elementStorage.Requests(), zero.Requests())...)
	physicalPointer, err := memorymarker.ElementToMemoryPointer(context, owner.children, source, element, elementPointer)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	layout, physicalElement, err := memorymarker.Layout(context, owner.children, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	raw, err := pointermarker.Operation(context, tsoniccore.SymbolToRawPointer, []api.TypeEmission{physicalElement}, []api.ExpressionEmission{physicalPointer, layout})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	word, err := model.WordType(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	count := func(member sliceruntime.Member) tsgo.Expression {
		operation := api.TargetIntrinsicNumber
		if model.WordBytes() == 8 {
			operation = api.TargetIntrinsicBigInt
		}
		query := factory.CallExpression(factory.PropertyAccessExpression(parameter, nil,
			factory.Identifier(sliceruntime.MemberName(member)), tsgo.NodeFlagsNone), nil, nil, nil, tsgo.NodeFlagsNone)
		return factory.AsExpression(factory.CallExpression(operation.Expression(factory), nil, nil, []tsgo.Expression{query}, tsgo.NodeFlagsNone), word.Value())
	}
	result := factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.DataMember), nil,
			factory.IndexedAccessTypeNode(storage.Value(), factory.LiteralTypeNode(factory.StringLiteral(descriptor.DataMember, tsgo.TokenFlagsNone))), raw.Value()),
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.LengthMember), nil, word.Value(), count(sliceruntime.MemberSourceLength)),
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.CapacityMember), nil, word.Value(), count(sliceruntime.MemberSourceCapacity)),
	}, true)
	body := api.DirectExpression(result, api.CombineRequests(raw.Requests(), word.Requests())...)
	body, err = api.NewExpressionEmission(raw.Before(), body.Value(), body.Requests())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return descriptorConversion(context, name, logical, storage, projected, body)
}

func (owner Owner) sliceFromDescriptor(context api.Context, source ast.Node, model descriptorvalue.Model, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	slice, element, ok := slicevalue.Source(model.SourceType())
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	logical, err := owner.children.RepresentedType(context, source, slice)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storage, err := model.StorageType(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	dataName, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	parameter := factory.Identifier(name)
	member := func(field string) tsgo.Expression {
		return factory.PropertyAccessExpression(parameter, nil, factory.Identifier(field), tsgo.NodeFlagsNone)
	}
	location, err := memorymarker.ElementLocation(context, owner.children, source, element, api.DirectExpression(factory.Identifier(dataName)))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	from, err := context.Names().Runtime(api.RuntimeSliceFromRegion, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementStorage, err := owner.ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	nilValue, err := owner.Zero(context, source, slice)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	panicReference, err := context.Names().Runtime(api.RuntimePanic, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	undefined := factory.VoidExpression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))
	invalid := factory.BinaryExpression(nil,
		factory.BinaryExpression(nil, member(descriptor.LengthMember), nil, factory.BinaryOperatorToken(tsgo.BinaryOperatorExclamationEqualsEqualsToken), model.ZeroWord(factory)),
		nil, factory.BinaryOperatorToken(tsgo.BinaryOperatorBarBarToken),
		factory.BinaryExpression(nil, member(descriptor.CapacityMember), nil, factory.BinaryOperatorToken(tsgo.BinaryOperatorExclamationEqualsEqualsToken), model.ZeroWord(factory)))
	body := []tsgo.Statement{factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(factory.Identifier(dataName), nil, nil, member(descriptor.DataMember)),
	}, tsgo.NodeFlagsConst)), factory.IfStatement(factory.BinaryExpression(nil, factory.Identifier(dataName), nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken), undefined), factory.Block(append([]tsgo.Statement{
		factory.IfStatement(invalid, factory.ExpressionStatement(panicruntime.Call(factory, panicReference.Name(),
			factory.StringLiteral("nil slice descriptor has nonzero extent", tsgo.TokenFlagsNone))), nil),
	}, append(nilValue.Before(), factory.ReturnStatement(nilValue.Value()))...), true), nil)}
	body = append(body, location.Before()...)
	result := factory.CallExpression(from.Expression(factory), nil, []tsgo.TypeNode{elementStorage.Value()}, []tsgo.Expression{
		memoryview.Pointer(factory, elementStorage.Value(), location.Value(), factory.NumericLiteral("0", tsgo.TokenFlagsNone)), member(descriptor.LengthMember), member(descriptor.CapacityMember),
	}, tsgo.NodeFlagsNone)
	converted, err := api.NewExpressionEmission(body, result, api.CombineRequests(location.Requests(), from.Requests(), elementStorage.Requests(), nilValue.Requests(), panicReference.Requests()))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	restored, err := descriptorConversion(context, name, storage, logical, value, converted)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return slicevalue.Wrap(context, model.SourceType(), restored)
}

func descriptorConversion(context api.Context, name string, inputType, outputType api.TypeEmission, input, body api.ExpressionEmission) (api.ExpressionEmission, error) {
	factory := context.Factory()
	transform := factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
		factory.ParameterDeclaration(nil, nil, factory.Identifier(name), nil, inputType.Value(), nil),
	}, outputType.Value(), factory.EqualsGreaterThanToken(), factory.Block(append(body.Before(), factory.ReturnStatement(body.Value())), true))
	return api.NewExpressionEmission(input.Before(), context.Factory().CallExpression(context.Factory().ParenthesizedExpression(transform), nil, nil,
		[]tsgo.Expression{input.Value()}, tsgo.NodeFlagsNone), api.CombineRequests(inputType.Requests(), outputType.Requests(), input.Requests(), body.Requests()))
}
