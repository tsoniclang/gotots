package storage

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Type(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	facet api.GenericRepresentationFacet,
) (api.TypeEmission, error) {
	switch facet {
	case api.GenericRepresentationStorage:
		return context.Values().StorageType(context, source, sourceType)
	case api.GenericRepresentationContainerStorage:
		return context.ContainerStorage().ContainerStorageType(
			context,
			source,
			sourceType,
		)
	default:
		return api.TypeEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic storage facet is invalid",
		}
	}
}

func Convert(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	facet api.GenericRepresentationFacet,
	direction api.GenericStorageDirection,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	switch {
	case facet == api.GenericRepresentationStorage &&
		direction == api.GenericStorageDirectionTo:
		return context.Values().ToStorage(
			context,
			source,
			sourceType,
			value,
		)
	case facet == api.GenericRepresentationStorage &&
		direction == api.GenericStorageDirectionFrom:
		return context.Values().FromStorage(
			context,
			source,
			sourceType,
			value,
		)
	case facet == api.GenericRepresentationContainerStorage &&
		direction == api.GenericStorageDirectionTo:
		return context.ContainerStorage().ToContainerStorage(
			context,
			source,
			sourceType,
			value,
		)
	case facet == api.GenericRepresentationContainerStorage &&
		direction == api.GenericStorageDirectionFrom:
		return context.ContainerStorage().FromContainerStorage(
			context,
			source,
			sourceType,
			value,
		)
	default:
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic storage projection is invalid",
		}
	}
}
