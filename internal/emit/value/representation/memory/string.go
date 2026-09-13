package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	descriptor "github.com/tsoniclang/gotots/internal/emit/runtime/memorydescriptor"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	runtimestring "github.com/tsoniclang/gotots/internal/emit/runtime/stringvalue"
	"github.com/tsoniclang/gotots/internal/emit/stringvalue"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	descriptorvalue "github.com/tsoniclang/gotots/internal/emit/value/memorydescriptor"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) stringToDescriptor(context api.Context, source ast.Node, model descriptorvalue.Model, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	if defined, ok := definedtype.ResolveBasic(model.SourceType()); ok {
		var err error
		value, err = defined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	logical, err := owner.children.RepresentedType(context, source, types.Typ[types.String])
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
	data, err := stringvalue.Member(context, runtimestring.DataMember, api.DirectExpression(parameter))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	layout, element, err := memorymarker.Layout(context, owner.children, source, types.Typ[types.Uint8])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	raw, err := pointermarker.Operation(context, tsoniccore.SymbolToRawPointer, []api.TypeEmission{element}, []api.ExpressionEmission{data, layout})
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	word, err := model.WordType(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	count, err := stringvalue.Member(context, runtimestring.SourceLengthMember, api.DirectExpression(parameter))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	conversion := api.TargetIntrinsicNumber
	if model.WordBytes() == 8 {
		conversion = api.TargetIntrinsicBigInt
	}
	length := factory.AsExpression(factory.CallExpression(conversion.Expression(factory), nil, nil, []tsgo.Expression{count.Value()}, tsgo.NodeFlagsNone), word.Value())
	result := factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.DataMember), nil,
			factory.IndexedAccessTypeNode(storage.Value(), factory.LiteralTypeNode(factory.StringLiteral(descriptor.DataMember, tsgo.TokenFlagsNone))), raw.Value()),
		factory.PropertyAssignment(nil, factory.Identifier(descriptor.LengthMember), nil, word.Value(), length),
	}, true)
	body, err := api.NewExpressionEmission(raw.Before(), result, api.CombineRequests(raw.Requests(), word.Requests(), count.Requests()))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return descriptorConversion(context, name, logical, storage, value, body)
}

func (owner Owner) stringFromDescriptor(context api.Context, source ast.Node, model descriptorvalue.Model, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	logical, err := owner.children.RepresentedType(context, source, types.Typ[types.String])
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
	member := func(field string) tsgo.Expression {
		return factory.PropertyAccessExpression(factory.Identifier(name), nil, factory.Identifier(field), tsgo.NodeFlagsNone)
	}
	data := factory.Identifier(dataName)
	location, err := memorymarker.ElementLocation(context, owner.children, source, types.Typ[types.Uint8], api.DirectExpression(data))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	empty, err := owner.Zero(context, source, types.Typ[types.String])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	panicReference, err := context.Names().Runtime(api.RuntimePanic, api.ImportPhaseValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	undefined := factory.VoidExpression(factory.NumericLiteral("0", tsgo.TokenFlagsNone))
	body := []tsgo.Statement{factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(data, nil, nil, member(descriptor.DataMember)),
	}, tsgo.NodeFlagsConst)), factory.IfStatement(factory.BinaryExpression(nil, data, nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsEqualsEqualsToken), undefined), factory.Block(append([]tsgo.Statement{
		factory.IfStatement(factory.BinaryExpression(nil, member(descriptor.LengthMember), nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorExclamationEqualsEqualsToken), model.ZeroWord(factory)),
			factory.ExpressionStatement(panicruntime.Call(factory, panicReference.Name(), factory.StringLiteral("nil string descriptor has nonzero extent", tsgo.TokenFlagsNone))), nil),
	}, append(empty.Before(), factory.ReturnStatement(empty.Value()))...), true), nil)}
	elementStorage, err := owner.ContainerStorageType(context, source, types.Typ[types.Uint8])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	region := api.DirectExpression(memoryview.Pointer(factory, elementStorage.Value(), location.Value(), factory.NumericLiteral("0", tsgo.TokenFlagsNone)),
		api.CombineRequests(location.Requests(), elementStorage.Requests())...)
	restored, err := stringvalue.Construct(context, runtimestring.FromRegionMember, region, api.DirectExpression(member(descriptor.LengthMember)))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	converted, err := api.NewExpressionEmission(append(body, restored.Before()...), restored.Value(), api.CombineRequests(restored.Requests(), empty.Requests(), panicReference.Requests()))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result, err := descriptorConversion(context, name, storage, logical, value, converted)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if defined, ok := definedtype.ResolveBasic(model.SourceType()); ok {
		return defined.Wrap(context, result)
	}
	return result, nil
}
