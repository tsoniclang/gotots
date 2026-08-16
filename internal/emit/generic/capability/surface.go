package capability

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildSurface(
	context api.Context,
	children api.ChildEmitter,
	artifact *api.GeneratedArtifact,
	modifiers []tsgo.ModifierLike,
) (tsgo.Statement, []api.RootRequest, error) {
	signature, selection, ok := artifact.GenericCapability()
	if !ok {
		return nil, nil, invariant(
			context,
			"generated artifact is not a generic capability",
		)
	}
	if sourceType, facet, direction, storage :=
		api.GenericStorageOperationType(selection, signature); storage {
		return buildStorageSurface(
			context,
			children,
			artifact.TargetName(),
			modifiers,
			sourceType,
			facet,
			direction,
		)
	}
	target, modifiers, resultType, err := buildCallableSurface(
		context,
		children,
		signature,
		artifact.Placement(),
		modifiers,
	)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().FunctionDeclaration(
		modifiers,
		nil,
		context.Factory().Identifier(artifact.TargetName()),
		nil,
		target.Parameters(),
		resultType,
		nil,
	), target.Requests(), nil
}

func buildCallableSurface(
	context api.Context,
	children api.ChildEmitter,
	signature *types.Signature,
	placement api.GeneratedArtifactPlacement,
	modifiers []tsgo.ModifierLike,
) (callable.SignatureEmission, []tsgo.ModifierLike, tsgo.TypeNode, error) {
	signatureRole := api.RoleFileDeclaration
	if placement == api.GeneratedArtifactPlacementLexical {
		signatureRole = api.RoleLocalDeclaration
	}
	target, err := callable.EmitAdapter(
		context.WithRole(signatureRole),
		children,
		nil,
		signature,
	)
	if err != nil {
		return callable.SignatureEmission{}, nil, nil, err
	}
	resultType := target.Result()
	if context.IsCooperative() {
		modifiers = append(modifiers, context.Factory().AsyncKeyword())
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return target, modifiers, resultType, nil
}
