package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericstorage "github.com/tsoniclang/gotots/internal/emit/generic/storage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func inlineStorageCapability(
	context api.Context,
	children api.ChildEmitter,
	signature *types.Signature,
	selection api.GenericOperationSelection,
) (api.ExpressionEmission, bool, error) {
	sourceType, facet, direction, ok :=
		api.GenericStorageOperationType(selection, signature)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		nil,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	storage, err := genericstorage.Type(
		context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
		facet,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	parameterType := logical.Value()
	resultType := storage.Value()
	value := context.Factory().Identifier("$argument0")
	if direction == api.GenericStorageDirectionFrom {
		parameterType, resultType = resultType, parameterType
	}
	result, err := genericstorage.Convert(
		context.WithRole(api.RoleFunctionBody),
		nil,
		sourceType,
		facet,
		direction,
		api.DirectExpression(value),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	body := append(
		result.Before(),
		context.Factory().ReturnStatement(result.Value()),
	)
	return api.DirectExpression(
		context.Factory().ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				context.Factory().ParameterDeclaration(
					nil,
					nil,
					value,
					nil,
					parameterType,
					nil,
				),
			},
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().Block(body, true),
		),
		api.CombineRequests(
			logical.Requests(),
			storage.Requests(),
			result.Requests(),
		)...,
	), true, nil
}
