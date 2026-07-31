package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericstorage "github.com/tsoniclang/gotots/internal/emit/generic/storage"
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
	sourceType, facet, direction, ok :=
		api.GenericStorageOperationType(selection, signature)
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
	storage, err := genericstorage.Type(
		context.WithRole(api.RoleStorageType),
		nil,
		sourceType,
		facet,
	)
	if err != nil {
		return nil, nil, true, err
	}
	parameterType := logical.Value()
	resultType := storage.Value()
	value := context.Factory().Identifier("$0")
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
