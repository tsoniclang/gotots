package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
)

func (owner Owner) ContainerStorageType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return owner.genericStorageType(
			context,
			source,
			parameter,
			api.GenericRepresentationContainerStorage,
			api.RuntimeContainerStorageType,
		)
	}
	required, err := owner.RequiresStorageProjection(context, sourceType)
	if err != nil {
		return api.TypeEmission{}, err
	}
	if !required {
		return owner.children.RepresentedType(context, source, sourceType)
	}
	return owner.StorageType(context, source, sourceType)
}

func (owner Owner) ToContainerStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationToContainerStorage,
			[]types.Type{parameter},
			[]types.Type{parameter},
			[]api.ExpressionEmission{value},
		)
	}
	required, err := owner.RequiresStorageProjection(context, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !required {
		return value, nil
	}
	return owner.ToStorage(context, source, sourceType, value)
}

func (owner Owner) FromContainerStorage(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if parameter, generic := api.GenericTypeParameter(sourceType); generic {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationFromContainerStorage,
			[]types.Type{parameter},
			[]types.Type{parameter},
			[]api.ExpressionEmission{value},
		)
	}
	required, err := owner.RequiresStorageProjection(context, sourceType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !required {
		return value, nil
	}
	return owner.FromStorage(context, source, sourceType, value)
}
