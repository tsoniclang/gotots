package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) arrayToMemory(context api.Context, source ast.Node, model arrayvalue.RuntimeArray, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	logical, err := model.EmitType(context, owner.children, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	physical, err := model.MemoryStorageType(context, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parameter, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	resultName, err := context.Names().Temporary(api.TemporaryArrayConstruction)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexName, err := context.Names().Temporary(api.TemporaryArrayConstruction)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if context.TypesSizes().Sizeof(model.SourceType()) == 0 {
		zero, err := pointermarker.Operation(context, tsoniccore.SymbolDefaultValue, []api.TypeEmission{physical}, nil)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return descriptorConversion(context, parameter, logical, physical, value, zero)
	}
	element, err := owner.MemoryStorageType(context, source, model.ElementType())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	index := factory.Identifier(indexName)
	read, err := model.ApplyIndex(context, api.DirectExpression(factory.Identifier(parameter)), api.DirectExpression(index))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	converted, err := owner.ToMemoryStorage(context, source, model.ElementType(), read)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := factory.Identifier(resultName)
	extent := arrayvalue.ExtentLiteral(factory, model.SourceType().Underlying().(*types.Array))
	body := []tsgo.Statement{factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(result, nil, factory.ArrayTypeNode(element.Value()), factory.ArrayLiteralExpression(nil, false)),
	}, tsgo.NodeFlagsConst)), factory.ForStatement(factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(index, nil, nil, factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
	}, tsgo.NodeFlagsLet), factory.BinaryExpression(nil, index, nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorLessThanToken), extent),
		factory.PostfixUnaryExpression(index, tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken),
		factory.Block(append(converted.Before(), factory.ExpressionStatement(factory.CallExpression(
			factory.PropertyAccessExpression(result, nil, factory.Identifier("push"), tsgo.NodeFlagsNone), nil, nil,
			[]tsgo.Expression{converted.Value()}, tsgo.NodeFlagsNone))), true))}
	panicReference, err := context.Names().Runtime(api.RuntimePanic, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	body = append(body, factory.IfStatement(factory.BinaryExpression(nil,
		factory.PropertyAccessExpression(result, nil, factory.Identifier("length"), tsgo.NodeFlagsNone), nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorExclamationEqualsEqualsToken), extent),
		factory.ExpressionStatement(panicruntime.Call(factory, panicReference.Name(),
			factory.StringLiteral("physical array extent mismatch", tsgo.TokenFlagsNone))), nil))
	projected, err := api.NewExpressionEmission(body, factory.AsExpression(result, physical.Value()),
		api.CombineRequests(element.Requests(), converted.Requests(), panicReference.Requests()))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return descriptorConversion(context, parameter, logical, physical, value, projected)
}

func (owner Owner) arrayFromMemory(context api.Context, source ast.Node, model arrayvalue.RuntimeArray, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	physical, err := model.MemoryStorageType(context, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logical, err := model.EmitType(context, owner.children, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parameter, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if model.Length() == 0 {
		zero, err := model.Zero(context, owner.children, source)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return descriptorConversion(context, parameter, physical, logical, value, zero)
	}
	indexName, err := context.Names().Temporary(api.TemporaryArrayConstruction)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	element, err := owner.MemoryStorageType(context, source, model.ElementType())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := owner.ContainerStorageType(context, source, model.ElementType())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	factory := context.Factory()
	var index tsgo.Expression = factory.NumericLiteral("0", tsgo.TokenFlagsNone)
	if context.TypesSizes().Sizeof(model.ElementType()) != 0 {
		index = factory.CallExpression(api.TargetIntrinsicNumber.Expression(factory), nil, nil,
			[]tsgo.Expression{factory.Identifier(indexName)}, tsgo.NodeFlagsNone)
	}
	address, err := pointermarker.Operation(context, tsoniccore.SymbolAddressOf, []api.TypeEmission{element}, []api.ExpressionEmission{
		api.DirectExpression(factory.ElementAccessExpression(factory.Identifier(parameter), nil, index, tsgo.NodeFlagsNone)),
	})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	projected, err := memorymarker.ProjectElementPointer(context, source, model.ElementType(), address)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	pointerType, err := pointermarker.Type(context, stored, false)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	location := factory.ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
		factory.ParameterDeclaration(nil, nil, factory.Identifier(indexName), nil, memoryview.CountType(factory), nil),
	}, pointerType.Value(), factory.EqualsGreaterThanToken(), factory.Block(append(projected.Before(), factory.ReturnStatement(projected.Value())), true))
	region := api.DirectExpression(memoryview.Pointer(factory, stored.Value(), location, factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
		api.CombineRequests(projected.Requests(), pointerType.Requests())...)
	restored, err := model.FromRegion(context, owner.children, region)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return descriptorConversion(context, parameter, physical, logical, value, restored)
}
