package storage

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct{}

func (Owner) Name(
	context api.Context,
	variable *types.Var,
) (string, bool) {
	return context.AddressableStorageName(variable)
}

func (owner Owner) Read(
	context api.Context,
	variable *types.Var,
) (api.ExpressionEmission, bool, error) {
	name, selected := owner.Name(context, variable)
	if !selected {
		return api.ExpressionEmission{}, false, nil
	}
	value := api.DirectExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(name),
			nil,
			context.Factory().Identifier(pointerruntime.CellValueName),
			tsgo.NodeFlagsNone,
		),
	)
	restored, err := context.Values().FromStorage(
		context,
		nil,
		variable.Type(),
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return restored, true, nil
}

func (owner Owner) StoreTarget(
	context api.Context,
	variable *types.Var,
) (api.StoreTargetEmission, bool, error) {
	name, selected := owner.Name(context, variable)
	if !selected {
		return api.StoreTargetEmission{}, false, nil
	}
	target, err := api.NewCanonicalStorageTargetEmission(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(name),
			nil,
			context.Factory().Identifier(pointerruntime.CellValueName),
			tsgo.NodeFlagsNone,
		),
		variable.Type(),
		nil,
	)
	return target, true, err
}

func (Owner) Cell(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	targetType, err := children.RepresentedType(
		context.WithRole(api.RoleLocalType),
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := context.Values().ToStorage(
		context,
		source,
		sourceType,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		stored.Before(),
		pointerruntime.Cell(
			context.Factory(),
			reference.Name(),
			targetType.Value(),
			storageType.Value(),
			stored.Value(),
		),
		api.CombineRequests(
			stored.Requests(),
			targetType.Requests(),
			storageType.Requests(),
			reference.Requests(),
		),
	)
}

func (Owner) Requirement(
	context api.Context,
	variable *types.Var,
) (api.RootRequest, error) {
	return api.NewAddressableStorageRequest(
		context.ArtifactOwner(),
		variable,
	)
}
