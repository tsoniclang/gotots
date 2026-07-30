package declaration

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericstorage "github.com/tsoniclang/gotots/internal/emit/generic/storage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitStorageOperationType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	operation *api.GenericOperationContract,
) (api.TypeEmission, bool, error) {
	sourceType, facet, direction, ok := api.GenericStorageOperationType(
		operation.Selection(),
		operation.Signature(),
	)
	if !ok {
		return api.TypeEmission{}, false, nil
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		source,
		sourceType,
	)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	storage, err := genericstorage.Type(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		facet,
	)
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	parameterType := logical.Value()
	resultType := storage.Value()
	if direction == api.GenericStorageDirectionFrom {
		parameterType, resultType = resultType, parameterType
	}
	return api.DirectType(
		context.Factory().FunctionTypeNode(
			nil,
			[]tsgo.ParameterDeclaration{
				context.Factory().ParameterDeclaration(
					nil,
					nil,
					context.Factory().Identifier("$0"),
					nil,
					parameterType,
					nil,
				),
			},
			resultType,
		),
		api.CombineRequests(
			logical.Requests(),
			storage.Requests(),
		)...,
	), true, nil
}
