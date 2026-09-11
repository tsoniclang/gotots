package memory

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ElementLocation(context api.Context, children api.ChildEmitter, source ast.Node, element types.Type, data api.ExpressionEmission) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	layout, storage, err := Layout(context, children, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	abi, err := DataLayout(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stride := context.TypesSizes().Sizeof(types.NewArray(element, 2)) - context.TypesSizes().Sizeof(element)
	if stride < 0 {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	factory := context.Factory()
	offset := factory.BinaryExpression(nil, factory.CallExpression(api.TargetIntrinsicBigInt.Expression(factory), nil, nil,
		[]tsgo.Expression{factory.Identifier(name)}, tsgo.NodeFlagsNone), nil, factory.BinaryOperatorToken(tsgo.BinaryOperatorAsteriskToken),
		factory.BigIntLiteral(strconv.FormatInt(stride, 10)+"n", tsgo.TokenFlagsNone))
	offsetType, err := context.Names().TsonicCore(tsoniccore.SymbolInt128)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	offsetName, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	offsetValue := factory.Identifier(offsetName)
	raw, err := pointermarker.Operation(context, tsoniccore.SymbolOffsetRawPointer, nil, []api.ExpressionEmission{data, api.DirectExpression(offsetValue), abi})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	address, err := pointermarker.Operation(context, tsoniccore.SymbolReinterpretRawPointer, []api.TypeEmission{storage}, []api.ExpressionEmission{raw, layout})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := context.ContainerStorage().ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	pointer, err := pointermarker.Type(context, stored, false)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	addressName, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	panicReference, err := context.Names().Runtime(api.RuntimePanic, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	addressValue := factory.Identifier(addressName)
	projected, err := ProjectElementPointer(context, source, element, api.DirectExpression(addressValue))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	body := []tsgo.Statement{factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(offsetValue, nil, factory.TypeReferenceNode(offsetType.EntityName(factory), nil), offset),
	}, tsgo.NodeFlagsConst))}
	body = append(body, address.Before()...)
	body = append(body, factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(addressValue, nil, nil, address.Value()),
	}, tsgo.NodeFlagsConst)), factory.IfStatement(factory.BinaryExpression(nil, addressValue, nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken), factory.VoidExpression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))),
		factory.ExpressionStatement(panicruntime.Call(factory, panicReference.Name(), factory.StringLiteral("nil element location", tsgo.TokenFlagsNone))), nil))
	body = append(body, projected.Before()...)
	body = append(body, factory.ReturnStatement(projected.Value()))
	return api.DirectExpression(factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
		factory.ParameterDeclaration(nil, nil, factory.Identifier(name), nil, memoryview.CountType(factory), nil),
	}, pointer.Value(), factory.EqualsGreaterThanToken(), factory.Block(body, true)),
		api.CombineRequests(pointer.Requests(), address.Requests(), projected.Requests(), panicReference.Requests(), offsetType.Requests())...), nil
}

func PointerRegion(context api.Context, children api.ChildEmitter, source ast.Node, element types.Type, pointer api.ExpressionEmission) (api.ExpressionEmission, error) {
	stored, err := ToStoragePointer(context, children, source, element, pointer)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	layout, storage, err := Layout(context, children, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	data, err := pointermarker.Operation(context, tsoniccore.SymbolToRawPointer, []api.TypeEmission{storage}, []api.ExpressionEmission{stored, layout})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Temporary(api.TemporaryAddressOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	dataValue := factory.Identifier(name)
	location, err := ElementLocation(context, children, source, element, api.DirectExpression(dataValue))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	region, err := context.Names().Runtime(api.RuntimeStorageRegion, api.ImportPhaseType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementStorage, err := context.ContainerStorage().ContainerStorageType(context, source, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	undefined := factory.VoidExpression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))
	body := append(data.Before(), factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(dataValue, nil, nil, data.Value()),
	}, tsgo.NodeFlagsConst)))
	body = append(body, location.Before()...)
	return api.NewExpressionEmission(body, factory.ConditionalExpression(factory.BinaryExpression(nil, dataValue, nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken), undefined), factory.QuestionToken(), undefined,
		factory.ColonToken(), memoryview.Pointer(factory, elementStorage.Value(), location.Value(), factory.NumericLiteral("0", tsgo.TokenFlagsNone))),
		api.CombineRequests(data.Requests(), location.Requests(), region.Requests(), elementStorage.Requests()))
}
