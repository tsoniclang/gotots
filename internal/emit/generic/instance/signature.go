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

func ConcreteMethodExpressionProviderSignature(
	signature *types.Signature,
) (*types.Signature, error) {
	concrete, err := ConcreteCallableSignature(signature)
	if err != nil {
		return nil, err
	}
	if concrete.Recv() != nil ||
		concrete.Params().Len() == 0 {
		return nil, &api.InvariantError{
			Role:   api.RoleCallableResult,
			Reason: "generic method-expression signature is invalid",
		}
	}
	parameters := make(
		[]*types.Var,
		0,
		concrete.Params().Len()-1,
	)
	for index := 1; index < concrete.Params().Len(); index++ {
		parameters = append(parameters, concrete.Params().At(index))
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		types.NewTuple(parameters...),
		concrete.Results(),
		concrete.Variadic(),
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
