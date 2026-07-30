package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (owner Owner) ContainerStorageType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return context.GenericParameterRepresentation(
			source,
			parameter,
			api.GenericRepresentationStorage,
		)
	}
	selection, err := owner.PointerRepresentation(
		context,
		types.NewPointer(sourceType),
		false,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return owner.PointerStorageType(
		context,
		source,
		sourceType,
		selection,
	)
}

func (owner Owner) PointerStorageType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	selection api.PointerRepresentationObservation,
) (api.TypeEmission, error) {
	if !selection.Representation().Valid() {
		return api.TypeEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer storage representation is invalid",
		}
	}
	if selection.Representation() ==
		api.PointerRepresentationCarrierCanonical {
		storage, err := owner.StorageType(context, source, sourceType)
		if err != nil {
			return api.TypeEmission{}, err
		}
		return api.DirectType(
			storage.Value(),
			api.CombineRequests(
				storage.Requests(),
				selection.Requests(),
			)...,
		), nil
	}
	logical, err := owner.children.RepresentedType(
		context,
		source,
		sourceType,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(
		logical.Value(),
		api.CombineRequests(
			logical.Requests(),
			selection.Requests(),
		)...,
	), nil
}

func (owner Owner) ToContainerStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, generic := api.GenericTypeParameter(sourceType); generic {
		return owner.ToStorage(context, source, sourceType, value)
	}
	selection, err := owner.PointerRepresentation(
		context,
		types.NewPointer(sourceType),
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return owner.ToPointerStorage(
		context,
		source,
		sourceType,
		selection,
		value,
	)
}

func (owner Owner) ToPointerStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	selection api.PointerRepresentationObservation,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !selection.Representation().Valid() {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer storage representation is invalid",
		}
	}
	var err error
	if selection.Representation() ==
		api.PointerRepresentationCarrierCanonical {
		value, err = owner.ToStorage(context, source, sourceType, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return api.NewExpressionEmission(
		value.Before(),
		value.Value(),
		api.CombineRequests(
			value.Requests(),
			selection.Requests(),
		),
	)
}

func (owner Owner) FromContainerStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, generic := api.GenericTypeParameter(sourceType); generic {
		return owner.FromStorage(context, source, sourceType, value)
	}
	selection, err := owner.PointerRepresentation(
		context,
		types.NewPointer(sourceType),
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return owner.FromPointerStorage(
		context,
		source,
		sourceType,
		selection,
		value,
	)
}

func (owner Owner) FromPointerStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	selection api.PointerRepresentationObservation,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !selection.Representation().Valid() {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer storage representation is invalid",
		}
	}
	var err error
	if selection.Representation() ==
		api.PointerRepresentationCarrierCanonical {
		value, err = owner.FromStorage(context, source, sourceType, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return api.NewExpressionEmission(
		value.Before(),
		value.Value(),
		api.CombineRequests(
			value.Requests(),
			selection.Requests(),
		),
	)
}
