package generic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	parameter, ok := api.GenericTypeParameter(sourceType)
	if ok {
		name, available := context.GenericParameterName(parameter)
		if !available {
			return api.TypeEmission{}, true,
				api.Unsupported(context, api.CategoryType, source)
		}
		return api.DirectType(
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(name),
				nil,
			),
		), true, nil
	}
	if _, defined := definedtype.Resolve(sourceType); defined {
		return api.TypeEmission{}, false, nil
	}
	object, arguments, instantiated := instantiatedType(sourceType)
	if !instantiated {
		return api.TypeEmission{}, false, nil
	}
	reference, err := context.Names().TypeReference(object)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	targetArguments, argumentRequests, err :=
		genericinstance.EmitTypeArguments(
			context,
			children,
			source,
			object,
			arguments,
		)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	return api.DirectType(
		context.Factory().TypeReferenceNode(
			reference.EntityName(context.Factory()),
			targetArguments,
		),
		api.CombineRequests(
			reference.Requests(),
			argumentRequests,
		)...,
	), true, nil
}

func instantiatedType(
	sourceType types.Type,
) (*types.TypeName, *types.TypeList, bool) {
	if object, arguments, ok := directInstantiatedType(sourceType); ok {
		return object, arguments, true
	}
	return directInstantiatedType(types.Unalias(sourceType))
}

func directInstantiatedType(
	sourceType types.Type,
) (*types.TypeName, *types.TypeList, bool) {
	switch source := sourceType.(type) {
	case *types.Named:
		if source.TypeArgs().Len() == 0 ||
			source.Origin() == nil ||
			source.Origin().Obj() == nil {
			return nil, nil, false
		}
		return source.Origin().Obj(), source.TypeArgs(), true
	case *types.Alias:
		if source.TypeArgs().Len() == 0 ||
			source.Origin() == nil ||
			source.Origin().Obj() == nil {
			return nil, nil, false
		}
		return source.Origin().Obj(), source.TypeArgs(), true
	default:
		return nil, nil, false
	}
}
