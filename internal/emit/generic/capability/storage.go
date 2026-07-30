package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildStorageCapability(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	modifiers []tsgo.ModifierLike,
	signature *types.Signature,
	selection api.GenericOperationSelection,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	sourceType, ok := api.GenericStorageOperationType(selection, signature)
	if !ok {
		return nil, nil, false, nil
	}
	if context.IsCooperative() {
		return nil, nil, true, invariant(
			context,
			"storage capability cannot be cooperative",
		)
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleParameterType),
		nil,
		sourceType,
	)
	if err != nil {
		return nil, nil, true, err
	}
	storage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
	)
	if err != nil {
		return nil, nil, true, err
	}
	parameterType := logical.Value()
	resultType := storage.Value()
	value := context.Factory().Identifier("$0")
	var result api.ExpressionEmission
	if selection.Operation() == api.GenericOperationFromStorage {
		parameterType, resultType = resultType, parameterType
		result, err = context.Values().FromStorage(
			context.WithRole(api.RoleFunctionBody),
			nil,
			sourceType,
			api.DirectExpression(value),
		)
	} else {
		result, err = context.Values().ToStorage(
			context.WithRole(api.RoleFunctionBody),
			nil,
			sourceType,
			api.DirectExpression(value),
		)
	}
	if err != nil {
		return nil, nil, true, err
	}
	body := append(
		result.Before(),
		context.Factory().ReturnStatement(result.Value()),
	)
	return context.Factory().FunctionDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(artifact.TargetName()),
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
			context.Factory().Block(body, true),
		),
		api.CombineRequests(
			logical.Requests(),
			storage.Requests(),
			result.Requests(),
		),
		true,
		nil
}
