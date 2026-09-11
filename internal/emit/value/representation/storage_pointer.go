package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	memorymarker "github.com/tsoniclang/gotots/internal/emit/marker/memory"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) ProjectStoragePointer(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	return owner.projectStoredPointer(context, source, sourceType, pointer, false)
}

func (owner Owner) ProjectMemoryPointer(context api.Context, source ast.Node, sourceType types.Type, pointer api.ExpressionEmission) (api.ExpressionEmission, error) {
	return owner.projectStoredPointer(context, source, sourceType, pointer, true)
}

func (owner Owner) projectStoredPointer(context api.Context, source ast.Node, sourceType types.Type, pointer api.ExpressionEmission, physical bool) (api.ExpressionEmission, error) {
	required, err := owner.RequiresStorageProjection(context, sourceType)
	if physical {
		required, err = memorymarker.RequiresProjection(context, sourceType)
	}
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
	storageTypeOf := owner.StorageType
	toStored := owner.ToStorage
	fromStored := owner.FromStorage
	if physical {
		storageTypeOf, toStored, fromStored = owner.MemoryStorageType, owner.ToMemoryStorage, owner.FromMemoryStorage
	}
	storageType, err := storageTypeOf(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageName, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logicalName, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	fromStorage, err := fromStored(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		api.DirectExpression(context.Factory().Identifier(storageName)),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	toStorage, err := toStored(
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
