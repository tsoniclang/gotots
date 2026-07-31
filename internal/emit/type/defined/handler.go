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
	if model.Type().TypeArgs().Len() != 0 {
		arguments, requests, err = genericinstance.EmitTypeArguments(
			context.WithRole(api.RoleDefinedTypeArgument),
			children,
			source,
			model.TypeName(),
			model.Type().TypeArgs(),
		)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
	}
	_, profiled := context.GenericCallableProfile()
	if RequiresValueFacet(model.Type()) &&
		(profiled || model.Type().TypeArgs().Len() != 0) {
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
