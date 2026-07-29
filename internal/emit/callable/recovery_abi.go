package callable

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const RecoveryAuthorityName = "$go$recovery"

func RecoveryAuthorityParameter(
	context api.Context,
) (tsgo.ParameterDeclaration, []api.RootRequest, error) {
	reference, err := context.Names().Runtime(
		api.RuntimeRecovery,
		api.ImportPhaseType,
	)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(RecoveryAuthorityName),
		context.Factory().QuestionToken(),
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(reference.Name()),
			nil,
		),
		nil,
	), reference.Requests(), nil
}
