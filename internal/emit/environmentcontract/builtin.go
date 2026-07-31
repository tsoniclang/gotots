package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
)

func BuiltinDeclaration(
	context api.Context,
	children api.ChildEmitter,
	builtin *types.Builtin,
	signature *types.Signature,
) (api.DeclarationEmission, error) {
	if !unsafeoperation.Classify(builtin).Runtime() ||
		signature == nil ||
		signature.Recv() != nil ||
		signature.TypeParams().Len() != 0 ||
		signature.RecvTypeParams().Len() != 0 {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "environment builtin overload is invalid",
		}
	}
	context, err := context.WithSourceArtifactOwner(
		api.MustSourceArtifactOwner(builtin),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	target, err := callable.EmitEnvironmentContract(
		context,
		children,
		signature,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	name, err := context.Names().Declare(builtin)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		context.Factory().FunctionDeclaration(
			exportDeclare(context),
			nil,
			context.Factory().Identifier(name),
			nil,
			target.Parameters(),
			target.Result(),
			nil,
		),
		target.Requests()...,
	), nil
}
