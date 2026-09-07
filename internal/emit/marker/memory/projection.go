package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ToStoragePointer(context api.Context, children api.ChildEmitter, source ast.Node, pointee types.Type, pointer api.ExpressionEmission) (api.ExpressionEmission, error) {
	required, err := context.Values().RequiresStorageProjection(context, pointee)
	if err != nil || !required {
		return pointer, err
	}
	logical, err := children.RepresentedType(context.WithRole(api.RoleStorageType), source, pointee)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := context.Values().StorageType(context.WithRole(api.RoleStorageType), source, pointee)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logicalName, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storedName, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	toStorage, err := context.Values().ToStorage(context, source, pointee, api.DirectExpression(context.Factory().Identifier(logicalName)))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	fromStorage, err := context.Values().FromStorage(context, source, pointee, api.DirectExpression(context.Factory().Identifier(storedName)))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	project := func(parameter string, input api.TypeEmission, output api.TypeEmission, body api.ExpressionEmission) api.ExpressionEmission {
		return api.DirectExpression(context.Factory().ArrowFunction(nil, nil, []tsgo.ParameterDeclaration{
			context.Factory().ParameterDeclaration(nil, nil, context.Factory().Identifier(parameter), nil, input.Value(), nil),
		}, output.Value(), context.Factory().EqualsGreaterThanToken(), context.Factory().Block(
			append(body.Before(), context.Factory().ReturnStatement(body.Value())), true,
		)), api.CombineRequests(input.Requests(), output.Requests(), body.Requests())...)
	}
	return pointermarker.Operation(context, tsoniccore.SymbolProjectPointer, []api.TypeEmission{logical, stored}, []api.ExpressionEmission{
		pointer, project(logicalName, logical, stored, toStorage), project(storedName, stored, logical, fromStorage),
	})
}
