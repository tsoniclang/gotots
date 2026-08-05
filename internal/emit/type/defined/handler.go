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
	representation, err := model.Representation(context)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	if representation.Kind() ==
		api.DefinedValueRepresentationProviderCanonical {
		target, err := children.RepresentedType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			source,
			model.Underlying(),
		)
		return target, true, err
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
			api.TypeArgumentsFromGo(model.Type().TypeArgs()),
		)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
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
