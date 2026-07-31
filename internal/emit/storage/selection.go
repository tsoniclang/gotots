package storage

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericpointer "github.com/tsoniclang/gotots/internal/emit/generic/pointer"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	if value, handled, err := genericpointer.Load(
		context,
		nil,
		variable.Type(),
		api.DirectExpression(context.Factory().Identifier(name)),
	); handled || err != nil {
		return value, true, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(variable.Type()),
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		value := api.DirectExpression(
			context.Factory().Identifier(name),
			representation.Requests()...,
		)
		copied, err := context.Values().Transfer(
			context,
			nil,
			variable.Type(),
			variable.Type(),
			api.ValueTransferCopy,
			value,
		)
		return copied, true, err
	}
	value := api.DirectExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(name),
			nil,
			context.Factory().Identifier(pointerruntime.CellValueName),
			tsgo.NodeFlagsNone,
		),
	)
	restored, err := context.ContainerStorage().FromPointerStorage(
		context,
		nil,
		variable.Type(),
		representation,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	emission, err := api.NewExpressionEmission(
		restored.Before(),
		restored.Value(),
		api.CombineRequests(
			restored.Requests(),
			representation.Requests(),
		),
	)
	return emission, true, err
}

func (owner Owner) StoreTarget(
	context api.Context,
	variable *types.Var,
) (api.StoreTargetEmission, bool, error) {
	name, selected := owner.Name(context, variable)
	if !selected {
		return api.StoreTargetEmission{}, false, nil
	}
	if target, handled, err := genericpointer.StoreTarget(
		context,
		nil,
		variable.Type(),
		api.DirectExpression(context.Factory().Identifier(name)),
	); handled || err != nil {
		return target, true, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(variable.Type()),
		false,
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		target, err := api.NewStableIdentityStoreTargetEmission(
			api.DirectExpression(
				context.Factory().Identifier(name),
				representation.Requests()...,
			),
			variable.Type(),
		)
		return target, true, err
	}
	constructor := api.NewPropertyStoreTargetEmission
	if representation.Representation() ==
		api.PointerRepresentationCarrierCanonical {
		constructor = api.NewCanonicalStoragePropertyStoreTargetEmission
	}
	target, err := constructor(
		context.Factory(),
		api.DirectExpression(
			context.Factory().Identifier(name),
			representation.Requests()...,
		),
		pointerruntime.CellValueName,
		variable.Type(),
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	return target, true, err
}

func (Owner) Cell(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if cell, handled, err := genericpointer.Cell(
		context,
		source,
		sourceType,
		value,
	); handled || err != nil {
		return cell, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(sourceType),
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			api.CombineRequests(
				value.Requests(),
				representation.Requests(),
			),
		)
	}
	targetType, err := children.RepresentedType(
		context.WithRole(api.RoleLocalType),
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		representation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := context.ContainerStorage().ToPointerStorage(
		context,
		source,
		sourceType,
		representation,
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
			representation.Requests(),
		),
	)
}

func (Owner) Requirement(
	context api.Context,
	variable *types.Var,
) (api.RootRequest, error) {
	owner := context.ArtifactOwner()
	source, sourceOwned := owner.Source()
	_, functionOwned := source.(*types.Func)
	_, _, initializerOwned := owner.PackageInitializer()
	if !sourceOwned && !initializerOwned ||
		sourceOwned && !functionOwned {
		return api.RootRequest{}, &api.InvariantError{
			Role: context.Role(),
			Reason: fmt.Sprintf(
				"addressable variable %q at %s has no reconstructible artifact owner %q",
				variable.Name(),
				context.FileSet().Position(variable.Pos()),
				owner.Name(),
			),
		}
	}
	return api.NewAddressableStorageRequest(owner, variable)
}
