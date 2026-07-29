package instance

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func ConcreteCallableSignature(
	signature *types.Signature,
) (*types.Signature, error) {
	if signature == nil ||
		signature.TypeParams().Len() != 0 {
		return nil, &api.InvariantError{
			Role:   api.RoleCallableResult,
			Reason: "generic callable signature is invalid",
		}
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		signature.Params(),
		signature.Results(),
		signature.Variadic(),
	), nil
}

func ReceiverTypeArguments(sourceType types.Type) *types.TypeList {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		sourceType = pointer.Elem()
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok ||
		named.TypeParams().Len() == 0 ||
		named.TypeArgs().Len() != named.TypeParams().Len() {
		return nil
	}
	return named.TypeArgs()
}
