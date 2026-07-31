package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericpointer "github.com/tsoniclang/gotots/internal/emit/generic/pointer"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
)

func canonicalPointerTarget(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	pointer api.ExpressionEmission,
	element types.Type,
) (api.StoreTargetEmission, error) {
	if target, handled, err := genericpointer.StoreTarget(
		context,
		source,
		element,
		pointer,
	); handled || err != nil {
		return target, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		false,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		logical, err := children.RepresentedType(
			context.WithRole(api.RoleAssignmentTarget),
			source,
			element,
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		runtime, err := pointerRuntime(context)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		location, err := api.NewExpressionEmission(
			pointer.Before(),
			pointerruntime.Direct(
				context.Factory(),
				runtime.Name(),
				logical.Value(),
				pointer.Value(),
			),
			api.CombineRequests(
				pointer.Requests(),
				logical.Requests(),
				runtime.Requests(),
				representation.Requests(),
			),
		)
		if err != nil {
			return api.StoreTargetEmission{}, err
		}
		return api.NewStableIdentityStoreTargetEmission(location, element)
	}
	storage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		element,
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	receiver, err := api.NewExpressionEmission(
		pointer.Before(),
		pointerruntime.Dereference(
			context.Factory(),
			runtime.Name(),
			logical.Value(),
			storage.Value(),
			pointer.Value(),
		),
		api.CombineRequests(
			pointer.Requests(),
			logical.Requests(),
			storage.Requests(),
			runtime.Requests(),
			representation.Requests(),
		),
	)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationCarrierCanonical {
		return api.NewCanonicalStoragePropertyStoreTargetEmission(
			context.Factory(),
			receiver,
			pointerruntime.CellValueName,
			element,
		)
	}
	return api.NewPropertyStoreTargetEmission(
		context.Factory(),
		receiver,
		pointerruntime.CellValueName,
		element,
	)
}
