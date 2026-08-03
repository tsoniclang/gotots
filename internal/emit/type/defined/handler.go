package defined

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	model, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{}, false, nil
	}
	reference, err := context.Names().TypeReference(model.TypeName())
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	var arguments []tsgo.TypeNode
	var requests []api.RootRequest
	if RequiresValueFacet(model.Type()) {
		providerRequests, err := providerCallableRequests(context, model)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
		requests = append(requests, providerRequests...)
	}
	if model.Type().TypeArgs().Len() != 0 {
		arguments, requests, err = genericinstance.EmitTypeArguments(
			context.WithRole(api.RoleDefinedTypeArgument),
			children,
			source,
			model.TypeName(),
			api.TypeArgumentsFromGo(model.Type().TypeArgs()),
		)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
	}
	explicitValueFacet, err := requiresExplicitValueFacet(context, model)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	if explicitValueFacet {
		underlying, err := valueFacetType(
			context,
			children,
			source,
			model,
		)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
		arguments = append(arguments, underlying.Value())
		requests = append(requests, underlying.Requests()...)
	}
	target := context.Factory().TypeReferenceNode(
		reference.EntityName(context.Factory()),
		arguments,
	)
	return api.DirectType(
		target,
		api.CombineRequests(reference.Requests(), requests)...,
	), true, nil
}

func requiresExplicitValueFacet(
	context api.Context,
	model Model,
) (bool, error) {
	if !RequiresValueFacet(model.Type()) {
		return false, nil
	}
	if _, profiled := context.GenericCallableProfile(); profiled {
		return true, nil
	}
	if model.Type().TypeArgs().Len() != 0 {
		return true, nil
	}
	representation, err := context.Names().DefinedValueRepresentation(
		model.TypeName(),
	)
	if err != nil {
		return false, err
	}
	_, providerIdentity := representation.ProviderCallableEffect()
	return providerIdentity, nil
}

func valueFacetType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	model Model,
) (api.TypeEmission, error) {
	return children.RepresentedType(
		context.WithRole(api.RoleDefinedUnderlyingType),
		source,
		model.Underlying(),
	)
}
