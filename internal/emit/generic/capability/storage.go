package capability

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericstorage "github.com/tsoniclang/gotots/internal/emit/generic/storage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type storageSurface struct {
	parameter tsgo.ParameterDeclaration
	result    tsgo.TypeNode
	requests  []api.RootRequest
	value     tsgo.Identifier
}

func buildStorage(
	context api.Context,
	children api.ChildEmitter,
	name string,
	modifiers []tsgo.ModifierLike,
	sourceType types.Type,
	facet api.GenericRepresentationFacet,
	direction api.GenericStorageDirection,
) (tsgo.Statement, []api.RootRequest, error) {
	surface, err := makeStorageSurface(
		context,
		children,
		name,
		sourceType,
		facet,
		direction,
	)
	if err != nil {
		return nil, nil, err
	}
	result, err := genericstorage.Convert(
		context.WithRole(api.RoleFunctionBody),
		nil,
		sourceType,
		facet,
		direction,
		api.DirectExpression(surface.value),
	)
	if err != nil {
		return nil, nil, err
	}
	body := append(
		result.Before(),
		context.Factory().ReturnStatement(result.Value()),
	)
	return storageSurfaceDeclaration(
			context,
			name,
			modifiers,
			surface,
			context.Factory().Block(body, true),
		), api.CombineRequests(
			surface.requests,
			result.Requests(),
		), nil
}

func buildStorageSurface(
	context api.Context,
	children api.ChildEmitter,
	name string,
	modifiers []tsgo.ModifierLike,
	sourceType types.Type,
	facet api.GenericRepresentationFacet,
	direction api.GenericStorageDirection,
) (tsgo.Statement, []api.RootRequest, error) {
	surface, err := makeStorageSurface(
		context,
		children,
		name,
		sourceType,
		facet,
		direction,
	)
	if err != nil {
		return nil, nil, err
	}
	return storageSurfaceDeclaration(
		context,
		name,
		modifiers,
		surface,
		nil,
	), surface.requests, nil
}

func makeStorageSurface(
	context api.Context,
	children api.ChildEmitter,
	name string,
	sourceType types.Type,
	facet api.GenericRepresentationFacet,
	direction api.GenericStorageDirection,
) (storageSurface, error) {
	if name == "" || sourceType == nil || !facet.Valid() || !direction.Valid() {
		return storageSurface{}, invariant(context, "generic storage capability is invalid")
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		nil,
		sourceType,
	)
	if err != nil {
		return storageSurface{}, err
	}
	storage, err := genericstorage.Type(
		context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
		facet,
	)
	if err != nil {
		return storageSurface{}, err
	}
	parameterType := logical.Value()
	resultType := storage.Value()
	value := context.Factory().Identifier("$argument0")
	if direction == api.GenericStorageDirectionFrom {
		parameterType, resultType = resultType, parameterType
	}
	return storageSurface{
		parameter: context.Factory().ParameterDeclaration(
			nil,
			nil,
			value,
			nil,
			parameterType,
			nil,
		),
		result: resultType,
		requests: api.CombineRequests(
			logical.Requests(),
			storage.Requests(),
		),
		value: value,
	}, nil
}

func storageSurfaceDeclaration(
	context api.Context,
	name string,
	modifiers []tsgo.ModifierLike,
	surface storageSurface,
	body tsgo.Block,
) tsgo.FunctionDeclaration {
	return context.Factory().FunctionDeclaration(
		slices.Clone(modifiers),
		nil,
		context.Factory().Identifier(name),
		nil,
		[]tsgo.ParameterDeclaration{surface.parameter},
		surface.result,
		body,
	)
}
