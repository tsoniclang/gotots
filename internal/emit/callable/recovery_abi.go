package callable

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	RecoveryAuthorityName = "$go$recovery"
)

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
		nil,
		context.Factory().TypeReferenceNode(
			reference.EntityName(context.Factory()),
			nil,
		),
		nil,
	), reference.Requests(), nil
}
