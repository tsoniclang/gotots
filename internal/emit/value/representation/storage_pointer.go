package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) ProjectStoragePointer(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	required, err := owner.RequiresStorageProjection(context, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !required {
		return pointer, nil
	}
	logicalType, err := owner.children.RepresentedType(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := owner.StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageName := "$go$storage"
	logicalName := "$go$value"
	fromStorage, err := owner.FromStorage(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		api.DirectExpression(context.Factory().Identifier(storageName)),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	toStorage, err := owner.ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		api.DirectExpression(context.Factory().Identifier(logicalName)),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	fromArrow := conversionArrow(
		context,
		storageName,
		storageType.Value(),
		logicalType.Value(),
		fromStorage,
	)
	toArrow := conversionArrow(
		context,
		logicalName,
		logicalType.Value(),
		storageType.Value(),
		toStorage,
	)
	return pointermarker.Operation(
		context,
		tsoniccore.SymbolProjectPointer,
		[]api.TypeEmission{storageType, logicalType},
		[]api.ExpressionEmission{
			pointer,
			api.DirectExpression(fromArrow, fromStorage.Requests()...),
			api.DirectExpression(toArrow, toStorage.Requests()...),
		},
	)
}

func conversionArrow(
	context api.Context,
	parameter string,
	sourceType tsgo.TypeNode,
	targetType tsgo.TypeNode,
	value api.ExpressionEmission,
) tsgo.ArrowFunction {
	statements := append(
		value.Before(),
		context.Factory().ReturnStatement(value.Value()),
	)
	return context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(parameter),
			nil,
			sourceType,
			nil,
		)},
		targetType,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(statements, true),
	)
}
