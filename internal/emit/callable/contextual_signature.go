package callable

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func sourceCallableSignature(
	context api.Context,
	target *types.Signature,
) (*types.Signature, error) {
	owner, generic := context.GenericParameterOwner()
	if !generic {
		return target, nil
	}
	function, ok := owner.(*types.Func)
	source, signatureOK := functionSignature(function)
	if !ok || !signatureOK || target == nil {
		return nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "contextual callable owner has no source signature",
		}
	}
	contextual, contextualOK := context.TypesInfo().
		TypeOfObject(function).(*types.Signature)
	if !contextualOK || !types.Identical(contextual, target) {
		return target, nil
	}
	return source, nil
}
