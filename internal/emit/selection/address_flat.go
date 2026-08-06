package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
)

func projectFieldPathPointer(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	parentType types.Type,
	parent api.ExpressionEmission,
	fields []*types.Var,
) (api.ExpressionEmission, int, error) {
	if len(fields) < 2 {
		return api.ExpressionEmission{}, 0, nil
	}
	parentRepresentation, err := pointertype.Observe(
		context,
		types.NewPointer(parentType),
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, 0, err
	}
	if parentRepresentation.Representation() !=
		api.PointerRepresentationCarrierCanonical {
		return api.ExpressionEmission{}, 0, nil
	}
	parentLogical, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		parentType,
	)
	if err != nil {
		return api.ExpressionEmission{}, 0, err
	}
	parentStorage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		parentType,
		parentRepresentation,
	)
	if err != nil {
		return api.ExpressionEmission{}, 0, err
	}
	requests := api.CombineRequests(
		parent.Requests(),
		parentLogical.Requests(),
		parentStorage.Requests(),
		parentRepresentation.Requests(),
	)
	currentType := parentType
	members := make([]string, 0, len(fields))
	var fieldLogical api.TypeEmission
	var fieldStorage api.TypeEmission
	for _, field := range fields {
		if !fieldInType(currentType, field) {
			break
		}
		_, providerOwned, providerErr := providerboundary.StructField(
			context,
			currentType,
			field,
		)
		if providerErr != nil {
			return api.ExpressionEmission{}, 0, providerErr
		}
		if providerOwned {
			break
		}
		parent, err = joinNominalFieldCallableABI(
			context,
			currentType,
			field,
			parent,
		)
		if err != nil {
			return api.ExpressionEmission{}, 0, err
		}
		fieldRepresentation, observeErr := pointertype.Observe(
			context,
			types.NewPointer(field.Type()),
			true,
		)
		if observeErr != nil {
			return api.ExpressionEmission{}, 0, observeErr
		}
		fieldLogical, err = children.RepresentedType(
			context.WithRole(api.RoleFieldReceiver),
			source,
			field.Type(),
		)
		if err != nil {
			return api.ExpressionEmission{}, 0, err
		}
		fieldStorage, err = context.Values().StorageType(
			context.WithRole(api.RoleStorageType),
			source,
			field.Type(),
		)
		if err != nil {
			return api.ExpressionEmission{}, 0, err
		}
		name, nameErr := context.Names().Member(field)
		if nameErr != nil {
			return api.ExpressionEmission{}, 0, nameErr
		}
		members = append(members, name)
		requests = api.CombineRequests(
			requests,
			parent.Requests(),
			fieldLogical.Requests(),
			fieldStorage.Requests(),
			fieldRepresentation.Requests(),
		)
		if _, _, _, pointer := pointerType(field.Type()); pointer {
			break
		}
		if fieldRepresentation.Representation() !=
			api.PointerRepresentationCarrierCanonical {
			break
		}
		currentType = field.Type()
	}
	if len(members) < 2 {
		return api.ExpressionEmission{}, 0, nil
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, 0, err
	}
	feature, err := api.NewRuntimeFeatureRequest(api.RuntimePointerFieldPath)
	if err != nil {
		return api.ExpressionEmission{}, 0, err
	}
	result, err := api.NewExpressionEmission(
		parent.Before(),
		pointerruntime.Fields(
			context.Factory(),
			runtime.Name(),
			fieldLogical.Value(),
			fieldStorage.Value(),
			parentLogical.Value(),
			parentStorage.Value(),
			parent.Value(),
			members,
		),
		api.CombineRequests(
			requests,
			runtime.Requests(),
			[]api.RootRequest{feature},
		),
	)
	return result, len(members), err
}
